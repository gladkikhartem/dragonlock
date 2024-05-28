package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var fmu = newFastLockMutex()
var rnd = rand.New(rand.NewSource(time.Now().Unix()))

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
		ctx.Error("duration len is not in range 0~3", 400)
		return
	}
	wait, err := strconv.Atoi(string(timeout))
	if err != nil {
		ctx.Error("failed to parse timeout value "+err.Error(), 400)
		return
	}

	cid := acc + string([]byte{0}) + id
	ch := make(chan bool)
	handle, ok := fmu.Lock(cid, ch, wait)
	if !ok {
		ctx.Error("lock timed out", 400)
		return
	}

	go func() {
		t := time.NewTimer(time.Second * time.Duration(dur))
		select {
		case <-t.C:
			fmu.UnlockTimeout(cid, ch) // try unlock by timeout
		case <-ch:
			t.Stop() // all is good
		}
	}()
	_, _ = ctx.Write([]byte(fmt.Sprint(handle)))
}

func FastUnlockHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	handle := int64(0)
	if len(ctx.Request.Body()) > 0 {
		handle, err = strconv.ParseInt(string(ctx.Request.Body()), 10, 64)
		if err != nil {
			ctx.Error(err.Error(), 400)
			return
		}
	}
	cid := acc + string([]byte{0}) + id
	ch := fmu.Unlock(cid, handle)
	if ch != nil {
		close(ch)
	}
}

// similar to keyed mutex, but allows for unlock timeouts
type fastLockMutex struct {
	c       *sync.Cond
	l       sync.Locker
	s       map[string]chan bool
	handles map[string]int64 // ensure that unlock is made by one who locked it
}

func newFastLockMutex() *fastLockMutex {
	l := sync.Mutex{}
	km := &fastLockMutex{c: sync.NewCond(&l), l: &l, s: make(map[string]chan bool), handles: map[string]int64{}}
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
	_, ok = km.s[key]
	return ok
}

func (km *fastLockMutex) Unlock(key string, handle int64) chan bool {
	km.l.Lock()
	defer km.l.Unlock()
	if handle != 0 && km.handles[key] != handle {
		return nil
	}
	ret := km.s[key]
	delete(km.s, key)
	km.c.Broadcast()
	return ret
}

func (km *fastLockMutex) UnlockTimeout(key string, ch chan bool) chan bool {
	km.l.Lock()
	defer km.l.Unlock()
	ret := km.s[key]
	if ret != ch {
		return nil
	}
	// unlock only if value is the same
	delete(km.s, key)
	km.c.Broadcast()
	return ret
}

func (km *fastLockMutex) Lock(key string, ch chan bool, wait int) (int64, bool) {
	start := time.Now().Unix()
	handle := rnd.Int63()
	km.l.Lock()
	defer km.l.Unlock()
	for km.locked(key) {
		km.c.Wait()
	}
	if int(time.Now().Unix()-start) > wait {
		return 0, false
	}
	km.s[key] = ch
	km.handles[key] = handle
	return handle, true
}
