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
	Version int64
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
		Data:    v.Value,
		Version: v.Version, // TODO: rename to sequence
	}
	d, err := dv.MarshalMsg(nil)
	if err != nil {
		return err
	}
	// log.Printf("save %v size: %v", v.Key, len(v.Value))
	return b.Set(compID(cd.KVPrefix, acc, v.Key), d, pebble.NoSync)
}

func handleKVGet(acc string, b *pebble.Batch, key string, res *Response) error {
	d, closer, err := b.Get(compID(cd.KVPrefix, acc, key))
	if err != nil {
		if err != pebble.ErrNotFound {
			return err
		}
		res.KVGet = append(res.KVGet, KV{
			Key:     key,
			Version: 0, // 0 version
		})
	}
	defer closer.Close()
	var v cd.KV
	_, err = v.UnmarshalMsg(d)
	if err != nil {
		return err
	}
	res.KVGet = append(res.KVGet, KV{
		Key:     key,
		Value:   v.Data,
		Version: v.Version,
	})
	return nil
}

func handle(acc string, req Request) (Response, error) {
	var res Response

	lockOnly := len(req.IdempotencyIDs) == 0 &&
		len(req.Atomic) == 00 &&
		len(req.KVGet) == 0 &&
		len(req.KVSet) == 0

	b := store.db.NewIndexedBatch() // TODO: maybe normal batch will work too
	if req.UnlockID != "" || req.LockID != "" {
		if req.UnlockID == req.LockID { // extend lock
			err := memExtendLock(acc, req.LockID, req.Unlock, req.LockDur)
			if err != nil {
				return res, err
			}
		} else {
			if !lockOnly && req.UnlockID != "" { // unlock, but we should unlock only after successful write, so extend for now
				err := memExtendLock(acc, req.UnlockID, req.Unlock, 30)
				if err != nil {
					return res, err
				}
				err = b.Delete(compID(cd.LocksPrefix, acc, req.UnlockID), pebble.NoSync)
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
				if lockOnly {
					err = store.db.Set(compID(cd.LocksPrefix, acc, req.LockID), d, pebble.Sync)
					if err != nil {
						return res, fmt.Errorf(err.Error())
					}
				} else {
					err = b.Set(compID(cd.LocksPrefix, acc, req.LockID), d, pebble.NoSync)
					if err != nil {
						return res, fmt.Errorf(err.Error())
					}
				}
			}
		}
	}
	ukey := []byte(acc)

	if !lockOnly {
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
			if len(req.KVSet) > 0 {
				seqID := compID1(cd.VerSequencePrefix, acc)
				ver, err := GetInt64(seqID, b)
				if err != nil {
					return err
				}
				v := int64(1)
				if ver != nil {
					v = *ver
				}
				for _, val := range req.KVSet {
					val.Version = v
					v++
					err = handleKVSet(acc, b, val)
					if err != nil {
						return err
					}
				}
				err = SetInt64(seqID, v, b)
				if err != nil {
					panic(err)
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
	}
	if req.LockID != req.UnlockID && req.UnlockID != "" { // unlock
		err := memUnlock(acc, req.UnlockID, res.Lock)
		if err != nil {
			log.Print("failed to unlock after successful write")
		}
		if lockOnly {
			err = store.db.Delete(compID(cd.LocksPrefix, acc, req.UnlockID), pebble.Sync)
			if err != nil {
				return res, fmt.Errorf(err.Error())
			}
		}
	}
	for _, val := range req.KVSet {
		store.notifier(acc).NotifyVersion(val.Key, val.Version)
	}

	return res, nil
}

var ccc int64

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
		return
	}

	d, err := json.Marshal(res)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	ctx.Response.SetBody(d)
}

type WatchRequest struct {
	ID      string
	Version int64
}

func WatchHandler(ctx *fasthttp.RequestCtx) {
	acc, err := getAcc(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	var req WatchRequest
	err = json.Unmarshal(ctx.Request.Body(), &req)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	if req.ID == "" {
		ctx.Error("id to watch is empty", 400)
		return
	}
	kv, err := watcher(acc, req.ID, req.Version)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	d, err := json.Marshal(kv)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	ctx.Response.SetBody(d)
}

func watcher(acc string, key string, ver int64) (KV, error) {
	n := store.notifier(acc)
	var kv *KV
	err := store.Singleton([]byte(acc), func() error {
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
				kv = &KV{
					Key:     key,
					Value:   v.Data,
					Version: v.Version,
				}
				return nil
			}
		}
		n.Attach(key, ver)
		return nil
	})
	if err != nil {
		return KV{}, err
	}
	if kv != nil { // key changed in DB already
		return *kv, nil
	}
	retV := n.Listen(key, ver, 30)
	if retV == -1 { // timeout
		return KV{}, fmt.Errorf("no change")
	}
	d, closer, err := store.db.Get(compID(cd.KVPrefix, acc, key))
	if err != nil {
		return KV{}, err
	}
	defer closer.Close()

	var v cd.KV
	_, err = v.UnmarshalMsg(d)
	if err != nil {
		return KV{}, err
	}
	return KV{
		Key:     key,
		Value:   v.Data,
		Version: v.Version,
	}, nil
}
