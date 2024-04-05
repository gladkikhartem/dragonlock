package main

import (
	"strconv"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

// API that allows incrementing / decrementing counter
// The different between this and Sequence is that
// sequence always goes up by 1, and no duplicated are possible.
// But counter can be manually controlled
func AddCounterHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	args := ctx.Request.URI().QueryArgs()
	toadd := args.Peek("add")
	if len(toadd) > 12 || len(toadd) == 0 {
		ctx.Error("add is not in range 0~12", 400)
		return
	}
	add, err := strconv.Atoi(string(toadd))
	if err != nil {
		ctx.Error("failed to parse add "+err.Error(), 400)
		return
	}
	cid := compID(CounterPrefix, acc, id)
	newCtr := int64(0)
	err = store.Update(cid, func() error {
		ctr, err := GetInt64(cid, store.db)
		if err != nil {
			return err
		}
		if ctr == nil {
			v := int64(0)
			ctr = &v
		}
		*ctr += int64(add)
		newCtr = *ctr
		return SetInt64(cid, *ctr, store.db)
	})
	if err != nil {
		ctx.Error("err updating: "+err.Error(), 400)
		return
	}
	_, _ = ctx.Write([]byte(strconv.FormatInt(newCtr, 10)))
}

func DeleteCounterHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(CounterPrefix, acc, id)
	err = store.Update(cid, func() error {
		return store.db.Delete(cid, pebble.NoSync)
	})
	if err != nil {
		ctx.Error("err updating: "+err.Error(), 400)
		return
	}
}

func GetCounterHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(CounterPrefix, acc, id)
	seq, err := GetInt64(cid, store.db)
	if err != nil {
		ctx.Error("err gettting: "+err.Error(), 400)
		return
	}
	if seq == nil {
		ctx.Error("not found", 404)
		return
	}
	_, _ = ctx.Write([]byte(strconv.FormatInt(*seq, 10)))
}
