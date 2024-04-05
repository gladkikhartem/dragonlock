package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

//go:generate msgp
type KV struct {
	Version int64  `json:"v" msg:"v"`
	Data    []byte `json:"d" msg:"d"`
}

var ErrConflict = errors.New("version mismatch")

// This is a simple key-value database with versioning support
// Enough for basic needs with optimistic concurrency
func SetKVHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(SequentialIDPrefix, acc, id)
	args := ctx.Request.URI().QueryArgs()
	version := args.Peek("ver")
	if len(version) > 12 || len(version) == 0 {
		ctx.Error("version is not in range 0~6", 400)
		return
	}
	oldVer, err := strconv.Atoi(string(version))
	if err != nil {
		ctx.Error("failed to parse duration "+err.Error(), 400)
		return
	}

	var kv KV
	err = store.Update(cid, func() error {
		d, closer, err := store.db.Get(cid)
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		// no kv present - can set
		if err == pebble.ErrNotFound {
			kv.Data = ctx.PostBody()
			kv.Version = 1
			nd, err := kv.MarshalMsg(nil)
			if err != nil {
				return err
			}
			return store.db.Set(cid, nd, pebble.NoSync)
		}
		defer closer.Close()

		_, err = kv.UnmarshalMsg(d)
		if err != nil {
			return err
		}
		if kv.Version != int64(oldVer) {
			return ErrConflict
		}
		kv.Data = ctx.PostBody()
		kv.Version++

		nd, err := kv.MarshalMsg(nil)
		if err != nil {
			return err
		}
		return store.db.Set(cid, nd, pebble.NoSync)
	})
	if err == ErrConflict {
		ctx.SetStatusCode(409)
	} else if err != nil {
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
	cid := compID(SequentialIDPrefix, acc, id)
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
	var kv KV
	_, err = kv.UnmarshalMsg(d)
	if err != nil {
		ctx.Error("db decode err: "+err.Error(), 500)
		return
	}
	ctx.Response.Header.Set("version", fmt.Sprint(kv.Version))
	_, _ = ctx.Write(kv.Data)
}

func DeleteKVHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	args := ctx.Request.URI().QueryArgs()
	version := args.Peek("ver")
	if len(version) > 12 || len(version) == 0 {
		ctx.Error("version is not in range 0~6", 400)
		return
	}
	oldVer, err := strconv.Atoi(string(version))
	if err != nil {
		ctx.Error("failed to parse duration "+err.Error(), 400)
		return
	}
	cid := compID(SequentialIDPrefix, acc, id)

	var kv KV
	err = store.Update(cid, func() error {
		d, closer, err := store.db.Get(cid)
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		// kv already deleted - skip
		if err == pebble.ErrNotFound {
			return nil
		}
		defer closer.Close()
		_, err = kv.UnmarshalMsg(d)
		if err != nil {
			return err
		}
		if kv.Version != int64(oldVer) {
			return ErrConflict
		}
		return store.db.Delete(cid, pebble.NoSync)
	})
	if err != nil {
		ctx.Error("err deleting: "+err.Error(), 400)
		return
	}
}
