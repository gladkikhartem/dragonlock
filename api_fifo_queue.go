package main

// //go:generate msgp
// type FIFOQueue struct {
// 	MsgCounter int64   `json:"len" msg:"q"`
// 	Counter    int64   `json:"-" msg:"c"`
// 	Messages   []int64 `json:"-" msg:"m"`
// }

// var ErrNoMessages = errors.New("No messages")

// // This is a simple FIFO queue
// func PushFIFOHandler(ctx *fasthttp.RequestCtx) {
// 	acc, id, err := getAccID(ctx)
// 	if err != nil {
// 		ctx.Error(err.Error(), 400)
// 		return
// 	}
// 	cid := compID(SequentialIDPrefix, acc, id)
// 	var q FIFOQueue
// 	err = store.Update(cid, func() error {
// 		d, closer, err := store.db.Get(cid)
// 		if err != nil && err != pebble.ErrNotFound {
// 			return err
// 		}
// 		// unmarshal if queue exists
// 		if err != pebble.ErrNotFound {
// 			_, err = q.UnmarshalMsg(d)
// 			if err != nil {
// 				return err
// 			}
// 			defer closer.Close()
// 		}
// 		q.Counter++
// 		q.MsgCounter++
// 		q.Messages = append(q.Messages, q.Counter)

// 		id2 := comp2ID(cid, Int64ToByte(q.Counter))
// 		b := store.db.NewBatch()
// 		err = b.Set(id2, ctx.PostBody(), pebble.NoSync)
// 		if err != nil {
// 			return err
// 		}
// 		nd, err := q.MarshalMsg(nil)
// 		if err != nil {
// 			return err
// 		}
// 		err = b.Set(cid, nd, pebble.NoSync)
// 		if err != nil {
// 			return err
// 		}
// 		return b.Commit(pebble.NoSync)
// 	})
// 	if err == lockErr {
// 		ctx.SetStatusCode(409)
// 	} else if err != nil {
// 		ctx.Error("err updating: "+err.Error(), 400)
// 		return
// 	}
// }

// // This is a simple FIFO queue
// func PopFIFOHandler(ctx *fasthttp.RequestCtx) {
// 	acc, id, err := getAccID(ctx)
// 	if err != nil {
// 		ctx.Error(err.Error(), 400)
// 		return
// 	}
// 	cid := compID(SequentialIDPrefix, acc, id)
// 	var retMsg []byte
// 	var q FIFOQueue
// 	err = store.Update(cid, func() error {
// 		d, closer, err := store.db.Get(cid)
// 		if err != nil && err != pebble.ErrNotFound {
// 			return err
// 		}
// 		// unmarshal if queue exists
// 		if err != pebble.ErrNotFound {
// 			_, err = q.UnmarshalMsg(d)
// 			if err != nil {
// 				return err
// 			}
// 			defer closer.Close()
// 		}
// 		if q.MsgCounter == 0 {
// 			return ErrNoMessages
// 		}
// 		id2 := comp2ID(cid, Int64ToByte(q.Messages[0]))
// 		out, closer, err := store.db.Get(id2)
// 		if err != nil {
// 			return err
// 		}
// 		retMsg = make([]byte, len(out))
// 		copy(retMsg, out)
// 		closer.Close()

// 		q.MsgCounter--
// 		q.Messages = append(q.Messages[1:])

// 		b := store.db.NewBatch()
// 		err = b.Delete(id2, pebble.NoSync)
// 		if err != nil {
// 			return err
// 		}
// 		nd, err := q.MarshalMsg(nil)
// 		if err != nil {
// 			return err
// 		}
// 		err = b.Set(cid, nd, pebble.NoSync)
// 		if err != nil {
// 			return err
// 		}
// 		return b.Commit(pebble.NoSync)
// 	})
// 	if err == lockErr {
// 		ctx.SetStatusCode(409)
// 	} else if err != nil {
// 		ctx.Error("err updating: "+err.Error(), 400)
// 		return
// 	}
// 	ctx.Response
// }
