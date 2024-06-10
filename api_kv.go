package main

import (
	"clouddragon/cd"
	"errors"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

var ErrConflict = errors.New("version mismatch")

// No versioning needed for KV if we can just lock the data by key
func SetKVHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(cd.KVPrefix, acc, id)
	var kv cd.KV
	err = store.Update(cid, func() error {
		kv.Data = ctx.PostBody()
		nd, err := kv.MarshalMsg(nil)
		if err != nil {
			return err
		}
		return store.db.Set(cid, nd, pebble.NoSync)
	})
	if err != nil {
		ctx.Error("err updating: "+err.Error(), 400)
		return
	}
}

func GetKVHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(cd.KVPrefix, acc, id)
	d, closer, err := store.db.Get(cid)
	if err != nil {
		if err == pebble.ErrNotFound {
			ctx.Error("not locked", 404)
		} else {
			ctx.Error("db read err: "+err.Error(), 500)
		}
		return
	}
	defer closer.Close()
	var kv cd.KV
	_, err = kv.UnmarshalMsg(d)
	if err != nil {
		ctx.Error("db decode err: "+err.Error(), 500)
		return
	}
	_, _ = ctx.Write(kv.Data)
}

func DeleteKVHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(cd.KVPrefix, acc, id)
	err = store.Update(cid, func() error {
		return store.db.Delete(cid, pebble.NoSync)
	})
	if err != nil {
		ctx.Error("err deleting: "+err.Error(), 400)
		return
	}
}
