package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

//go:generate msgp
type Lock struct {
	Owner string `json:"o" msg:"o"`
	Till  int64  `json:"t" msg:"t"`
}

var lockErr = errors.New("locked")

// This API stores locks and allows lock owner to extend the lock
// Very useful if you want to make sure some script is executed in singleton
// curl -f -X POST "http://localhost:8081/db/1/lock/1?id=123&dur=30" && ...
// and now you know no other process will run this operation for next 30 seconds
func SetLockHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := compID(SequentialIDPrefix, acc, id)
	args := ctx.Request.URI().QueryArgs()
	owner := string(args.Peek("owner"))
	if len(owner) > 255 {
		ctx.Error("owner is not in range 0~255", 400)
		return
	}
	duration := args.Peek("dur")
	if len(duration) > 6 || len(duration) == 0 {
		ctx.Error("duration is not in range 0~6", 400)
		return
	}
	dur, err := strconv.Atoi(string(duration))
	if err != nil {
		ctx.Error("failed to parse duration "+err.Error(), 400)
		return
	}
	var l Lock
	err = store.Update(cid, func() error {
		d, closer, err := store.db.Get(cid)
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		// no lock present - can lock
		if err == pebble.ErrNotFound {
			l.Till = time.Now().Add(time.Second * time.Duration(dur)).Unix()
			l.Owner = owner
			nd, err := l.MarshalMsg(nil)
			if err != nil {
				return err
			}
			return store.db.Set(cid, nd, pebble.NoSync)
		}
		defer closer.Close()

		_, err = l.UnmarshalMsg(d)
		if err != nil {
			return err
		}
		// already timed out - can lock
		if time.Now().Unix() > l.Till {
			l.Till = time.Now().Add(time.Second * time.Duration(dur)).Unix()
			l.Owner = owner
			nd, err := l.MarshalMsg(nil)
			if err != nil {
				return err
			}
			return store.db.Set(cid, nd, pebble.NoSync)
		}
		// allow owner to extend lock
		if len(l.Owner) != 0 && l.Owner == owner {
			l.Till = time.Now().Add(time.Second * time.Duration(dur)).Unix()
			nd, err := l.MarshalMsg(nil)
			if err != nil {
				return err
			}
			return store.db.Set(cid, nd, pebble.NoSync)
		}
		// lock exists - can't lock more
		return lockErr
	})
	if err == lockErr {
		ctx.SetStatusCode(409)
	} else if err != nil {
		ctx.Error("err updating: "+err.Error(), 400)
		return
	}
	d, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}
	_, _ = ctx.Write(d)
}

func GetLockHandler(ctx *fasthttp.RequestCtx) {
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
	var l Lock
	_, err = l.UnmarshalMsg(d)
	if err != nil {
		ctx.Error("db decode err: "+err.Error(), 500)
		return
	}
	if time.Now().Unix() > l.Till {
		ctx.Error("not locked", 404)
		return
	}
	_, _ = ctx.Write(d)
}

func DeleteLockHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	args := ctx.Request.URI().QueryArgs()
	owner := string(args.Peek("owner"))
	if len(owner) > 255 {
		ctx.Error("owner is not in range 0~255", 400)
		return
	}
	cid := compID(SequentialIDPrefix, acc, id)
	var l Lock
	err = store.Update(cid, func() error {
		d, closer, err := store.db.Get(cid)
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		// lock deleted - skip
		if err == pebble.ErrNotFound {
			return nil
		}
		defer closer.Close()
		_, err = l.UnmarshalMsg(d)
		if err != nil {
			return err
		}
		// already timed out
		if time.Now().Unix() > l.Till {
			return store.db.Delete(cid, pebble.NoSync)
		}
		// owner deletes his lock
		if l.Owner != "" && owner == l.Owner {
			return store.db.Delete(cid, pebble.NoSync)
		}
		// lock exists - can't delete
		return lockErr

	})
	if err != nil {
		ctx.Error("err deleting: "+err.Error(), 400)
		return
	}
}
