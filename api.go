package main

import (
	"log"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

type CntOp struct {
	Key    string
	Add    int
	Delete bool
}
type RatelimitOp struct {
	Key string
	Add int
}

type QueueMsg struct {
	ID   string
	Data []byte
}

type EnqueueOp struct {
	Queue string
	Vals  QueueMsg
}

type DequeueOp struct {
	Queue string
	Count int
	Ack   []int64 //IDs of previous messages to acknowledge
}

type KV struct {
	Key   string
	Value []byte
}

type LockOp struct {
	Wait         int
	Dur          int
	HandleRepair bool // assume repair status as "fail" and let app cleanup on it's side
}

// Bulk requests only make sense if Locks are not blocking
// so request will succeed only if lock was successful

// Stupidly lock by DB(parititon) for KV & Queue operations!!!!

type Request struct {
	Unlock   int64 // if both lock & unlock = extend the lock
	LockWait int
	LockDur  int
	// lock is made by DB ID. But it's only required for KV & Queue operations
	// Other operations can perform faster without locks

	// all operations are performed/written only if lock/unlock were successful
	Cnt       []CntOp       // no locks - works ok
	Ratelimit []RatelimitOp // no locks - works ok
	Enqueue   []EnqueueOp   // no locks - works ok - new messages are added

	Dequeue []DequeueOp // lock required (dequeue + lock => dequeue + unlock)
	// whole Partition is locked (KV and Queues should have different lock spaces,
	// Queues should be tied to whole DB
	// KV should be always manual - i.e. lock by specific key, prefix or whatever
	//
	// During request - we may not know the partition we need, so queues are
	// kind of generic - we probably want to have some consistent maglev hash over workers
	// to distribute partitions between them.

	KVSet []KV // lock optional (get + lock =>  set + unlock)
	// lock desired if we want to ensure corretness
	// during queue processing kv will be already locked

	KVGet          []string // no locks
	KVDelete       []string // no locks
	IdempotencyIDs []string // no locks
}

type CntRes struct {
	Key   string
	Value int64
}

type DequeueRes struct {
	Dequeued []QueueMsg
}

// Expectation is that all API
type Response struct {
	DB         string // partition used
	Lock       int64  // id to unlock the lock
	LockResult string // locked, not_locked, repair.
	// during repair - any actions are not performed - to allow app to handle repair.
	// if repair is not needed - app can simply resend requires with
	// handleID to extend the lock and apply operations
	Dequeue []DequeueRes
	KVGet   []string
	Cnt     []CntRes
}

func RequestHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	log.Print(acc, id)
	req := Request{}
	ukey := []byte(acc)

	if req.LockDur != 0 { // TODO:  add repair status handling
		// in-memory lock logic
		// optionally - extend lock if needed by doing Unlock & Lock withing mutex
	} else {
		// we can definitely use simpler kmutex implementation here
	}

	var res Response
	// all updates for one DB are performed sequentially, so high-contention
	// updates to same db will not block each other
	err = store.Update(ukey, func() error {
		b := store.db.NewBatch()

		// Counter logic
		for _, cnt := range req.Cnt {
			if cnt.Delete {
				err := b.Delete([]byte(cnt.Key), pebble.NoSync) // TODO: KEY
				if err != nil {
					panic(err)
				}
			}
			if cnt.Add != 0 {
				ctr, err := GetInt64([]byte(cnt.Key), store.db)
				if err != nil {
					return err
				}
				if ctr == nil {
					v := int64(0)
					ctr = &v
				}
				*ctr += int64(cnt.Add)
				res.Cnt = append(res.Cnt, CntRes{
					Value: *ctr,
				})
				return SetInt64([]byte(cnt.Key), *ctr, store.db)
			} else {
				ctr, err := GetInt64([]byte(cnt.Key), store.db)
				if err != nil {
					return err
				}
				if ctr == nil {
					v := int64(0)
					ctr = &v
				}
				res.Cnt = append(res.Cnt, CntRes{
					Value: *ctr,
				})
			}
		}

		// Queue logic
		// for _, v := range req.Enqueue {

		// }
		// for _, v := range req.Dequeue {

		// }
		// for _, v := range req.Set {

		// }
		// for _, v := range req.Get {

		// }
		// TODO: add lock / unlock to batch
		return b.Commit(pebble.NoSync)
	})
	if req.Unlock != 0 {
		// Unlock in memory only after write has succeeded, since otherwise other process can
		// start writing to DB before batch is completed
	}
	if err != nil {
		ctx.Error("err updating: "+err.Error(), 400)
		return
	}
}
