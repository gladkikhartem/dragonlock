package main

import (
	"clouddragon/cd"
	"strconv"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

func NextSequenceHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(cd.SequentialIDPrefix, acc, id)
	newSeq := int64(0)
	err = store.Update(cid, func() error {
		seq, err := GetInt64(cid, store.db)
		if err != nil {
			return err
		}
		if seq == nil {
			v := int64(0)
			seq = &v
		}
		*seq++
		newSeq = *seq
		return SetInt64(cid, *seq, store.db)
	})
	if err != nil {
		ctx.Error("err updating: "+err.Error(), 400)
		return
	}
	_, _ = ctx.Write([]byte(strconv.FormatInt(newSeq, 10)))
}

func DeleteSequenceHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(cd.SequentialIDPrefix, acc, id)
	err = store.Update(cid, func() error {
		return store.db.Delete(cid, pebble.NoSync)
	})
	if err != nil {
		ctx.Error("err updating: "+err.Error(), 400)
		return
	}
}

func GetSequenceHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(cd.SequentialIDPrefix, acc, id)
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
