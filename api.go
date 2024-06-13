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
	IfEq *int64 // conditional update (for ex. safely reset counter)
}

type KV struct {
	Key     string
	Value   json.RawMessage
	Delete  bool
	Version int64 `json:"-"` // Only for internal use
}

type EnqueueOp struct {
	Queue    string
	Messages []json.RawMessage
	Counter  int64
}

type Request struct {
	LockWait int
	LockDur  int
	LockID   string

	UnlockID string
	Unlock   int64 // if both lockid & unlockid = extend the lock

	IdempotencyIDs []string // TODO: configure Idempotency Records  TTL (merge function)
	Atomic         []AtomicOp
	KVSet          []*KV
	KVGet          []string
}

type AtomicRes struct {
	Key                string `json:"k,omitempty"`
	Old                int64  `json:"old,omitempty"`
	New                int64  `json:"new,omitempty"`
	PreconditionFailed bool   `json:"f"`
}

// Expectation is that all API
type Response struct {
	Lock int64 `json:"l,omitempty"` // id to unlock the lock. 0 - lock failed
	// LockRepair bool  `json:"rep,omitempty"` // previous lock timed out
	// during repair - any actions are not performed - to allow app to handle repair.
	// if repair is not needed - app can simply resend requires with
	// handleID to extend the lock and apply operations
	KVGet  []KV        `json:"kv,omitempty"`
	Atomic []AtomicRes `json:"atm,omitempty"`
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
			Key: op.Key,
			Old: *val - op.Add,
			New: *val,
		})
		return SetInt64(id, *val, b)
	}
	if op.Set != 0 {
		if op.IfEq != nil && *op.IfEq != *val {
			res.Atomic = append(res.Atomic, AtomicRes{
				Key:                op.Key,
				Old:                *val,
				PreconditionFailed: true,
			})
			return nil
		}
		res.Atomic = append(res.Atomic, AtomicRes{
			Key:                op.Key,
			Old:                *val,
			New:                op.Set,
			PreconditionFailed: true,
		})
		val = &op.Set
		return SetInt64(id, *val, b)
	}
	return fmt.Errorf("empty atomic request")
}

func handleKVSet(acc string, b *pebble.Batch, v *KV) error {
	if v.Delete {
		return b.Delete(compID(cd.KVPrefix, acc, v.Key), pebble.NoSync)
	}
	dv := cd.KV{
		Data: v.Value,
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
			Key: key,
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
	err := store.Singleton(ukey, func() error {
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
		for _, v := range req.KVGet {
			err := handleKVGet(acc, b, v, &res)
			if err != nil {
				return err
			}
		}
		for _, val := range req.KVSet {
			ver, err := GetInt64(compID1(cd.VerSequencePrefix, acc), b)
			if err != nil {
				return err
			}
			v := int64(1)
			if ver != nil {
				v = *ver
			}
			val.Version = v
			err = handleKVSet(acc, b, val)
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
	// notify all listeners (if any) for kv change
	for _, val := range req.KVSet {
		store.notifier(acc).SetVersion(val.Key, val.Version)
	}

	// notify all listeners (if any) for atomic counter change
	for _, val := range res.Atomic {
		// TODO: different key space for Counters
		store.notifier(acc).SetVersion(val.Key, val.New)
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

// Service boots up, checks the version local with version on server
// if version differs - he gets update
// if version doesn't differ - request blocks for specified duration
// request with version 0 (get latest value) blocks only if value is missing

// If service wants to update config (for example add itself to potential member list)
// It makes "lock" on the KV it wants to update, reads it, then performs the update
// then unlocks.
//
// If we have 10 nodes and they report their status every second - 10 req/sec
// is nothing for server to handle
//
// Each node can become leader when needed by locking the KV
// # Leader can react to membership changes and update real config that everyone follows
//
// Nodes can in return update their status (last config version ) so that leader
// knows which version they are using
//
// Cool scheme, but it will be good if that all can be make easier for client implementation
func watcher(acc string, key string, ver int64) (int64, error) {
	n := store.notifier(acc)
	attached := false
	_ = store.Singleton([]byte(acc), func() error {
		d, closer, err := store.db.Get(compID(cd.KVPrefix, acc, key))
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		if err == nil {
			defer closer.Close()
			var v cd.KV
			_, err = v.UnmarshalMsg(d)
			if err != nil {
				return err
			}
			if v.Version != ver {
				return nil
			}
		}
		n.Attach(key, ver)
		attached = true
		return nil
	})
	if attached {
		_ = n.Listen(acc, ver, 30)
	}
	// TODO: return value if version different
	return 0, nil
}
