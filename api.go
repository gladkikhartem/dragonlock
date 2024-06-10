package main

import (
	"log"

	"github.com/valyala/fasthttp"
)

type AddOp struct {
	Key string
	Val int
}

type QueueMsg struct {
}

type EnqueueOp struct {
	Queue string
	Vals  QueueMsg
}

type DequeueOp struct {
	Queue string
	Count int
	Ack   []int64 //IDs of messages to acknowledge
}

type KV struct {
	Key   string
	Value []byte
}

type LockOp struct {
	Key  string
	Wait int
	Dur  int
}

type Request struct {
	Lock   LockOp // multiple locks are not allowed
	Unlock int64  // if both lock & unlock = extend the lock

	// all operations are performed/written only if lock/unlock were successful
	Add     []AddOp
	Enqueue []EnqueueOp
	Dequeue []DequeueOp
	Set     []KV
	Get     []string
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

	if req.Lock.Key != "" {
		// in-memory lock logic
		// optionally - extend lock if needed by doing Unlock & Lock withing mutex
	} else {
		// if lock wasn't requested - we still need to perform in-memory
		// lock to ensure that only one update is performed per key at a time
		//
		// we can definitely use simpler kmutex implementation here
	}

	// all updates for one DB are performed sequentially, so high-contention
	// updates to same db will not block each other
	err = store.Update(ukey, func() error {
		// b := store.db.NewBatch()
		// for _, add := range req.Add {
		// 	//
		// }
		// for _, v := range req.Enqueue {

		// }
		// for _, v := range req.Dequeue {

		// }
		// for _, v := range req.Set {

		// }
		// for _, v := range req.Get {

		// }
		// TODO: add lock / unlock to batch
		return nil
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
