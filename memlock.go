package main

import (
	"clouddragon/cd"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
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
		km := chooseLock(string(cid))
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

func chooseLock(id string) *fastLockMutex {
	h := fnv.New64a()
	h.Write([]byte(id))
	kid := h.Sum64()
	return fmu[kid%mCount]
}

func memLock(acc, id string, dur, wait int) (int64, error) {
	cid := acc + string([]byte{0}) + id
	handle, ok := chooseLock(cid).Lock(cid, dur, wait, 0)
	if !ok {
		return 0, cd.ErrNotLocked
	}
	return handle, nil
}

func memUnlock(acc, id string, handle int64) error {
	cid := acc + string([]byte{0}) + id
	ch, err := chooseLock(cid).Unlock(cid, handle)
	if err != nil {
		return err
	}
	if ch != nil {
		close(ch)
	}
	return nil
}

func memExtendLock(acc, id string, handle int64, dur int) bool {
	cid := acc + string([]byte{0}) + id
	return chooseLock(cid).extendLock(cid, handle, time.Now().Unix()+int64(dur))
}

// RWLock implementation:
// Broadcast() instead of Signal()
// Writer updates "lock" value & waits for unlock
// Readers wait for unlock
// all good and cool - but what about timeouts & lock extensions???
// 1. Don't allow to extend Rlock (only lock)
// 2. Timeouts will update readers count, so all handles will be stored inside

type FLock struct {
	ch     chan bool
	handle int64
	till   int64
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

func (km *fastLockMutex) extendLock(key string, handle int64, till int64) bool {
	km.l.Lock()
	defer km.l.Unlock()
	fl, ok := km.m[key]
	if !ok {
		return false
	}
	if fl.handle != handle {
		return false
	}
	fl.till = till
	km.m[key] = fl
	return true
}

func (km *fastLockMutex) Unlock(key string, handle int64) (chan bool, error) {
	km.l.Lock()
	defer km.l.Unlock()
	return km.unlock(key, handle)
}

func (km *fastLockMutex) unlock(key string, handle int64) (chan bool, error) {
	fl, found := km.m[key]
	if !found {
		return nil, nil
	}
	if handle != 0 && fl.handle != handle {
		return nil, fmt.Errorf("handle mismatch")
	}
	delete(km.m, key)
	km.c.Signal()
	return fl.ch, nil
}

func (km *fastLockMutex) UnlockTimeout(key string, till int64, ch chan bool) (chan bool, int64) {
	km.l.Lock()
	defer km.l.Unlock()
	fl := km.m[key]
	if fl.ch != ch {
		return nil, 0
	}
	if fl.till != till { // reschedule timer
		return nil, fl.till
	}
	// unlock only if value is the same
	delete(km.m, key)
	km.c.Signal()
	return fl.ch, 0
}

var handleCounter = int64(1)

func (km *fastLockMutex) Lock(key string, dur, wait int, oldHandle int64) (int64, bool) {
	km.l.Lock()
	defer km.l.Unlock()
	return km.lock(key, dur, wait, oldHandle)
}

func (km *fastLockMutex) lock(key string, dur, wait int, oldHandle int64) (int64, bool) {
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
	fl := FLock{
		ch:     ch,
		handle: handle,
		till:   time.Now().Unix() + int64(dur),
	}
	go func() {
		t := time.NewTimer(time.Second * time.Duration(dur))
		till := fl.till
		for {
			select {
			case <-t.C:
				retCh, newTill := km.UnlockTimeout(key, till, ch) // try unlock by timeout
				if newTill != till {
					t.Reset(time.Second * time.Duration(newTill-till))
					till = newTill
					continue
				}
				if retCh != nil {
					close(retCh)
				}
				break
			case <-ch:
				// closed due to success
				t.Stop()
				break
			}
		}
	}()
	km.m[key] = fl
	return handle, true
}

type FRWLock struct {
	// write lock
	ch     chan bool
	handle int64
	till   int64
	locked bool

	// read locks
	rhandles map[int64]chan bool
}

// similar to keyed mutex, but allows for unlock timeouts & RW logic
type fastRWMutex struct {
	c *sync.Cond
	l sync.Locker
	m map[string]FRWLock
}

func newfastRWMutex() *fastRWMutex {
	l := sync.Mutex{}
	km := &fastRWMutex{c: sync.NewCond(&l), l: &l, m: map[string]FRWLock{}}
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

func (km *fastRWMutex) locked(key string) bool {
	v, ok := km.m[key]
	if !ok {
		return false
	}
	return v.locked
}

func (km *fastRWMutex) rlocked(key string) bool {
	v, ok := km.m[key]
	if !ok {
		return false
	}
	return len(v.rhandles) > 0
}

func (km *fastRWMutex) Unlock(key string, handle int64) (chan bool, error) {
	km.l.Lock()
	defer km.l.Unlock()
	return km.unlock(key, handle)
}

func (km *fastRWMutex) RUnlock(key string, handle int64) (chan bool, error) {
	km.l.Lock()
	defer km.l.Unlock()
	return km.runlock(key, handle)
}

func (km *fastRWMutex) unlock(key string, handle int64) (chan bool, error) {
	fl, found := km.m[key]
	if !found {
		return nil, nil
	}
	if handle != 0 && fl.handle != handle {
		return nil, fmt.Errorf("handle mismatch")
	}
	fl.locked = false
	if len(fl.rhandles) == 0 { // if lock is empty - delete it
		delete(km.m, key)
	}
	km.c.Broadcast()
	return fl.ch, nil
}

func (km *fastRWMutex) runlock(key string, handle int64) (chan bool, error) {
	fl, found := km.m[key]
	if !found {
		return nil, nil
	}
	ch, ok := fl.rhandles[handle]
	if !ok {
		return nil, fmt.Errorf("handle mismatch")
	}
	if !fl.locked && len(fl.rhandles) == 0 { // if lock is empty - delete it
		delete(km.m, key)
	} else {
		delete(fl.rhandles, handle)
	}
	km.c.Broadcast()
	return ch, nil
}

// TODO: extend read lock, but only if no locker is waiting
func (km *fastRWMutex) UnlockTimeout(key string, till int64, handle int64) (chan bool, int64) {
	km.l.Lock()
	defer km.l.Unlock()
	fl := km.m[key]
	if fl.handle != handle {
		return nil, 0
	}
	if fl.till != till { // reschedule timer
		return nil, fl.till
	}
	fl.locked = false
	if len(fl.rhandles) == 0 { // if lock is empty - delete it
		delete(km.m, key)
	}
	km.c.Broadcast()
	return fl.ch, 0
}

func (km *fastRWMutex) RUnlockTimeout(key string, till int64, handle int64) (chan bool, int64) {
	km.l.Lock()
	defer km.l.Unlock()
	fl := km.m[key]
	ch, ok := fl.rhandles[handle]
	if !ok {
		return nil, 0
	}
	if !fl.locked && len(fl.rhandles) == 0 { // if lock is empty - delete it
		delete(km.m, key)
	} else {
		delete(fl.rhandles, handle)
	}
	km.c.Broadcast()
	return ch, 0
}

func (km *fastRWMutex) Lock(key string, dur, wait int, oldHandle int64) (int64, bool) {
	km.l.Lock()
	defer km.l.Unlock()
	return km.lock(key, dur, wait, oldHandle)
}

func (km *fastRWMutex) RLock(key string, dur, wait int, oldHandle int64) (int64, bool) {
	km.l.Lock()
	defer km.l.Unlock()
	return km.rlock(key, dur, wait, oldHandle)
}

func (km *fastRWMutex) lock(key string, dur, wait int, oldHandle int64) (int64, bool) {
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
	tl, ok := km.m[key]
	if ok {
		tl.locked = true
		km.m[key] = tl
	}

	// wait all readers to finish
	for km.rlocked(key) {
		// woke up by broadcast - i.e. lock operation timed out
		if wait == 0 || int(time.Now().Unix()-start) > wait {
			return 0, false
		}
		km.c.Wait()
	}

	// lock, but unlock this key automatically if expires
	ch := make(chan bool)
	fl := FRWLock{
		ch:       ch,
		handle:   handle,
		till:     time.Now().Unix() + int64(dur),
		rhandles: map[int64]chan bool{}, // TODO: map vs slice benchmark
	}
	go func() {
		t := time.NewTimer(time.Second * time.Duration(dur))
		till := fl.till
		for {
			select {
			case <-t.C:
				retCh, newTill := km.UnlockTimeout(key, till, handle) // try unlock by timeout
				if newTill != till {
					t.Reset(time.Second * time.Duration(newTill-till))
					till = newTill
					continue
				}
				if retCh != nil {
					close(retCh)
				}
				break
			case <-ch:
				// closed due to success
				t.Stop()
				break
			}
		}
	}()
	km.m[key] = fl
	return handle, true
}

func (km *fastRWMutex) rlock(key string, dur, wait int, oldHandle int64) (int64, bool) {
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

	fl := km.m[key]
	if fl.rhandles == nil {
		fl.rhandles = map[int64]chan bool{}
	}
	fl.rhandles[handle] = ch
	km.m[key] = fl
	go func() {
		t := time.NewTimer(time.Second * time.Duration(dur))
		till := fl.till
		for {
			select {
			case <-t.C:
				retCh, newTill := km.RUnlockTimeout(key, till, handle) // try unlock by timeout
				if newTill != till {
					t.Reset(time.Second * time.Duration(newTill-till))
					till = newTill
					continue
				}
				if retCh != nil {
					close(retCh)
				}
				break
			case <-ch:
				// closed due to success
				t.Stop()
				break
			}
		}
	}()
	return handle, true
}
