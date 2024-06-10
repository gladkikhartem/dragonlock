package main

import (
	"clouddragon/cd"
	"fmt"
	"hash/fnv"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

var fmu = []*fastLockMutex{}

func InitFastLocks() {
	for i := 0; i < mCount; i++ {
		fmu = append(fmu, newFastLockMutex())
	}
	iter, err := store.db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{cd.LocksPrefix},
		UpperBound: []byte{cd.LocksPrefix + 1},
	})
	if err != nil {
		panic(err)
	}
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		cid := fromCompID1(key)
		km := GetLock(string(cid))
		var f cd.Lock
		_, err := f.UnmarshalMsg(value)
		if err != nil {
			panic(err)
		}
		dur := f.Till - time.Now().Unix()
		if dur < 0 {
			err := store.db.Delete(key, pebble.NoSync)
			if err != nil {
				panic(err)
			}
			continue
		}
		// make sure that after reboot counter doesn't start with
		// number lower than any lock handle stored in
		if handleCounter < f.Handle {
			handleCounter = f.Handle + 1
		}
		_, ok := km.Lock(cid, int(dur), 0, f.Handle)
		if !ok {
			panic("lock should always work during startup")
		}
	}
}

func GetLock(id string) *fastLockMutex {
	h := fnv.New64a()
	h.Write([]byte(id))
	kid := h.Sum64()
	return fmu[kid%mCount]
}

func GetLockHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	cid := acc + string([]byte{0}) + id

	d, closer, err := store.db.Get(compID1(cd.LocksPrefix, cid))
	if err != nil {
		if err == pebble.ErrNotFound {
			ctx.Error(err.Error(), 404)
			return
		} else {
			ctx.Error(err.Error(), 500)
			return
		}
	}
	defer closer.Close()

	var f cd.Lock
	_, err = f.UnmarshalMsg(d)
	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}

}

func FastLockHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	duration := ctx.Request.URI().QueryArgs().Peek("dur")
	if len(duration) > 3 || len(duration) == 0 {
		ctx.Error("duration len is not in range 0~3", 400)
		return
	}
	dur, err := strconv.Atoi(string(duration))
	if err != nil {
		ctx.Error("failed to parse duration "+err.Error(), 400)
		return
	}
	timeout := ctx.Request.URI().QueryArgs().Peek("wait")
	if len(timeout) > 3 || len(timeout) == 0 {
		ctx.Error("wait len is not in range 0~3", 400)
		return
	}
	wait, err := strconv.Atoi(string(timeout))
	if err != nil {
		ctx.Error("failed to parse timeout value "+err.Error(), 400)
		return
	}

	handle, err := fastLockHandler(acc, id, dur, wait)
	if err == cd.ErrNotLocked {
		ctx.Error(err.Error(), 409)
		return
	}
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	_, _ = ctx.Write([]byte(`{"h": `))
	_, _ = ctx.Write([]byte(fmt.Sprint(handle)))
	_, _ = ctx.Write([]byte(`}`))
}

func fastLockHandler(acc, id string, dur, wait int) (int64, error) {
	cid := acc + string([]byte{0}) + id
	handle, ok := GetLock(cid).Lock(cid, dur, wait, 0)
	if !ok {
		return 0, cd.ErrNotLocked
	}
	l := cd.Lock{
		Handle: handle,
		Till:   time.Now().Unix() + int64(dur),
	}
	d, err := l.MarshalMsg(nil)
	if err != nil {
		return 0, err
	}
	return handle, store.db.Set(compID1(cd.LocksPrefix, cid), d, pebble.Sync)
}

func FastUnlockHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}

	handle := int64(0)
	h := ctx.Request.URI().QueryArgs().Peek("h")
	if len(h) > 0 {
		handle, err = strconv.ParseInt(string(h), 10, 64)
		if err != nil {
			ctx.Error(err.Error(), 400)
			return
		}
	}
	err = fastUnlockHandler(acc, id, handle)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
}

func fastUnlockHandler(acc, id string, handle int64) error {
	cid := acc + string([]byte{0}) + id
	ch, err := GetLock(cid).Unlock(cid, handle)
	if err != nil {
		return err
	}
	if ch != nil {
		close(ch)
	}
	return store.db.Delete(compID1(cd.LocksPrefix, cid), pebble.NoSync)
}

type FLock struct {
	ch     chan bool
	handle int64
}

// similar to keyed mutex, but allows for unlock timeouts
type fastLockMutex struct {
	c *sync.Cond
	l sync.Locker
	m map[string]FLock
}

func newFastLockMutex() *fastLockMutex {
	l := sync.Mutex{}
	km := &fastLockMutex{c: sync.NewCond(&l), l: &l, m: map[string]FLock{}}
	go func() {
		// wake up all locks to make sure that
		// some locks don't stuck forever waiting and can handle
		// timeout event

		// MEMORY LEAK. TODO: make a shutdown procedure for this.
		// For now it should be ok, since this is a singleton struct
		t := time.NewTicker(time.Second)
		for range t.C {
			km.l.Lock()
			km.c.Broadcast()
			km.l.Unlock()
		}
	}()
	return km
}

func (km *fastLockMutex) locked(key string) (ok bool) {
	_, ok = km.m[key]
	return ok
}

func (km *fastLockMutex) Unlock(key string, handle int64) (chan bool, error) {
	km.l.Lock()
	defer km.l.Unlock()
	fl, found := km.m[key]
	if !found {
		return nil, nil
	}
	log.Print(fl.handle, " ", handle)
	if handle != 0 && fl.handle != handle {
		return nil, fmt.Errorf("handle mismatch")
	}
	delete(km.m, key)
	km.c.Signal()
	return fl.ch, nil
}

func (km *fastLockMutex) UnlockTimeout(key string, ch chan bool) chan bool {
	km.l.Lock()
	defer km.l.Unlock()
	fl := km.m[key]
	if fl.ch != ch {
		return nil
	}
	// unlock only if value is the same
	delete(km.m, key)
	km.c.Signal()
	return fl.ch
}

var handleCounter = int64(1)

func (km *fastLockMutex) Lock(key string, dur, wait int, oldHandle int64) (int64, bool) {
	start := time.Now().Unix()
	handle := atomic.AddInt64(&handleCounter, 1)
	if oldHandle != 0 {
		handle = oldHandle
	}
	km.l.Lock()
	defer km.l.Unlock()
	for km.locked(key) {
		// woke up by broadcast - i.e. lock operation timed out
		if wait == 0 || int(time.Now().Unix()-start) > wait {
			return 0, false
		}
		km.c.Wait()
	}

	// lock, but unlock this key automatically if expires
	ch := make(chan bool)
	go func() {
		t := time.NewTimer(time.Second * time.Duration(dur))
		select {
		case <-t.C:
			km.UnlockTimeout(key, ch) // try unlock by timeout
		case <-ch:
			// closed due to success
		}
		t.Stop()
	}()
	km.m[key] = FLock{
		ch:     ch,
		handle: handle,
	}
	return handle, true
}
