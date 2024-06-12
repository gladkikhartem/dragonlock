package main

import (
	"clouddragon/cd"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
	json "github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

type AtomicOp struct {
	Key  string
	Add  int64
	Set  int64
	IfEq *int64 // conditional update
}

type QueueMsg struct {
	QID  string `json:"qid,omitempty"` // id of the queue
	Data string `json:"raw,omitempty"` // message data
}

type EnqueueOp struct {
	Queue string
	Vals  []QueueMsg
}

type DequeueOp struct {
	Queue string
	Count int
	Ack   []int64 //IDs of previous messages to acknowledge
}

type KV struct {
	Key   string
	Value string
}

type LockOp struct {
	Wait int
	Dur  int
}

// Bulk requests only make sense if Locks are not blocking
// so request will succeed only if lock was successful

// Stupidly lock by DB(parititon) for KV & Queue operations!!!!

type Request struct {
	// lock can be requested separately or together with operations
	// if lock/unlock was requested - all operations are performed
	// only after lock/unlock were successful

	LockWait int
	LockDur  int
	LockID   string

	UnlockID string
	Unlock   int64 // if both lockid & unlockid = extend the lock

	IdempotencyIDs []string
	Atomic         []AtomicOp
	Enqueue        []EnqueueOp
	KVSet          []KV
	KVGet          []string
}

type AtomicRes struct {
	Key                string `json:"k,omitempty"`
	Value              int64  `json:"v,omitempty"`
	PreconditionFailed bool   `json:"f"`
}

// Expectation is that all API
type Response struct {
	Lock int64 `json:"l,omitempty"` // id to unlock the lock. 0 - lock failed
	// LockRepair bool  `json:"rep,omitempty"` // previous lock timed out
	// during repair - any actions are not performed - to allow app to handle repair.
	// if repair is not needed - app can simply resend requires with
	// handleID to extend the lock and apply operations
	Dequeue []QueueMsg  `json:"dq,omitempty"`
	KVGet   []KV        `json:"kv,omitempty"`
	Atomic  []AtomicRes `json:"atm,omitempty"`
}

func handleIdempotency(acc string, b *pebble.Batch, id string) error {
	_, closer, err := b.Get(compID(cd.IdempotencyPrefix, acc, id))
	if err != nil && err == pebble.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	closer.Close()
	return fmt.Errorf("duplicate request: " + id)
}

func handleAtomic(acc string, b *pebble.Batch, op AtomicOp, res *Response) error {
	id := compID(cd.AtomicPrefix, acc, op.Key)
	val, err := GetInt64(id, b)
	if err != nil {
		return err
	}
	if val == nil {
		v := int64(0)
		val = &v
	}
	if op.Add != 0 {
		*val += op.Add
		res.Atomic = append(res.Atomic, AtomicRes{
			Key:   op.Key,
			Value: *val,
		})
		return SetInt64(id, *val, b)
	}
	if op.Set != 0 {
		if op.IfEq != nil && *op.IfEq != *val {
			res.Atomic = append(res.Atomic, AtomicRes{
				Key:                op.Key,
				Value:              *val,
				PreconditionFailed: true,
			})
			return nil
		}
		val = &op.Set
		return SetInt64(id, *val, b)
	}
	return fmt.Errorf("empty atomic request")
}

//	<- dequeue |front|-----------------|back|  <- enqueue
//
// simple FIFO queue impelementation. All messages are processed sequentially.
// Performance can be improved simply by hashing with some ID.
// Since this is simple - can be done by clients and not needed here right now
func handleEnqueue(acc string, b *pebble.Batch, msg EnqueueOp) error {
	if len(msg.Vals) < 1 {
		return nil
	}
	var q cd.QueueMeta
	qid := compID(cd.QueueMetaPrefix, acc, msg.Queue)
	d, closer, err := b.Get(qid)
	if err != nil {
		if err != pebble.ErrNotFound {
			return err
		}
		q = cd.QueueMeta{
			Front: 1,
			Back:  0,
		}
	}
	defer closer.Close()
	if err == nil {
		_, err = q.UnmarshalMsg(d)
		if err != nil {
			return err
		}
	}
	for _, v := range msg.Vals {
		q.Back += 1
		err = b.Set(compID(cd.QueueMsgPrefix, acc, fmt.Sprint(q.Back)), []byte(v.Data), pebble.NoSync)
		if err != nil {
			return err
		}
	}
	d, err = q.MarshalMsg(nil)
	if err != nil {
		return err
	}
	return b.Set(qid, d, pebble.NoSync)
}

func handleKVSet(acc string, b *pebble.Batch, v KV) error {
	dv := cd.KV{
		Data: v.Value,
	}
	if v.Value == "" {
		return b.Delete(compID(cd.KVPrefix, acc, v.Key), pebble.NoSync)
	}
	d, err := dv.MarshalMsg(nil)
	if err != nil {
		return err
	}
	return b.Set(compID(cd.KVPrefix, acc, v.Key), d, pebble.NoSync)
}

func handleKVGet(acc string, b *pebble.Batch, key string, res *Response) error {
	d, closer, err := b.Get(compID(cd.KVPrefix, acc, key))
	if err != nil {
		if err != pebble.ErrNotFound {
			return err
		}
		res.KVGet = append(res.KVGet, KV{
			Key:   key,
			Value: "",
		})
	}
	defer closer.Close()
	var v cd.KV
	_, err = v.UnmarshalMsg(d)
	if err != nil {
		return err
	}
	res.KVGet = append(res.KVGet, KV{
		Key:   key,
		Value: v.Data,
	})
	return nil
}

func handle(acc string, req Request) (Response, error) {
	var res Response
	b := store.db.NewBatch()
	if req.UnlockID == req.LockID { // extend lock
		ok := memExtendLock(acc, req.LockID, req.Unlock, req.LockDur)
		if !ok {
			return res, fmt.Errorf("unable to extend lock")
		}
	} else {
		if req.UnlockID != "" { // unlock, but we should unlock only after successful write, so extend for now
			ok := memExtendLock(acc, req.UnlockID, req.Unlock, 30)
			if !ok {
				return res, fmt.Errorf("unable to extend lock")
			}
			err := b.Delete(compID(cd.LocksPrefix, acc, req.UnlockID), pebble.NoSync)
			if err != nil {
				return res, fmt.Errorf(err.Error())
			}
		}
		if req.LockID != "" { // lock
			newHandle, err := memLock(acc, req.LockID, req.LockDur, req.LockWait)
			if err != nil {
				return res, fmt.Errorf(err.Error())
			}
			res.Lock = newHandle
			c := cd.Lock{
				Handle: newHandle,
				Till:   time.Now().Add(time.Second * time.Duration(req.LockDur)).Unix(),
			}
			d, err := c.MarshalMsg(nil)
			if err != nil {
				return res, fmt.Errorf(err.Error())
			}
			err = b.Set(compID(cd.LocksPrefix, acc, req.LockID), d, pebble.NoSync)
			if err != nil {
				return res, fmt.Errorf(err.Error())
			}
		}
	}

	ukey := []byte(acc)
	// all updates for single key are performed sequentially, but flushed to
	// disk together. See store.Update for more info
	err := store.Update(ukey, func() error {
		for _, v := range req.IdempotencyIDs {
			err := handleIdempotency(acc, b, v)
			if err != nil {
				return err
			}
		}
		for _, v := range req.Atomic {
			err := handleAtomic(acc, b, v, &res)
			if err != nil {
				return err
			}
		}
		for _, v := range req.Enqueue {
			err := handleEnqueue(acc, b, v)
			if err != nil {
				return err
			}
		}
		for _, v := range req.KVGet {
			err := handleKVGet(acc, b, v, &res)
			if err != nil {
				return err
			}
		}
		for _, v := range req.KVSet {
			err := handleKVSet(acc, b, v)
			if err != nil {
				return err
			}
		}
		return b.Commit(pebble.NoSync)
	})
	if err != nil {
		if req.LockID != req.UnlockID && req.LockID != "" { // locked, but request failed - unlock
			err := memUnlock(acc, req.LockID, res.Lock)
			if err != nil {
				log.Print("failed to unlock after lock + failed write")
			}
		}
		return res, fmt.Errorf("err updating: " + err.Error())
	}
	if req.LockID != req.UnlockID && req.UnlockID != "" { // unlock
		err = memUnlock(acc, req.UnlockID, res.Lock)
		if err != nil {
			log.Print("failed to unlock after successful write")
		}
	}
	return res, nil
}

func RequestHandler(ctx *fasthttp.RequestCtx) {
	acc, err := getAcc(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	var req Request
	err = json.Unmarshal(ctx.Request.Body(), &req)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	res, err := handle(acc, req)
	if err != nil {
		ctx.Error(err.Error(), 400)
	}

	d, err := json.Marshal(res)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	ctx.Response.SetBody(d)
}
