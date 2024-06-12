// This is an implementation of "read-modify-write" storage that guarantees
// callers that all updates to an object happen one after another and
// after Update function returns - update was written to disk.
//
// '|_' - Start,  U- Update Logic   '_|' - End,  '_' - waiting,  '^' - data is flushed
// Request #1 ------|U_____________________|-------
// Request #1 --------------|U_____________|-------
// Request #2 --------------|_U____________|-------
// Request #3 --------------|__U___________|-------
// Flush Loop -----------------------------^-------
//
// We keep a global mutex by ID in RAM that makes sure that all updates
// happen in sequential manner. When we have multiple updates simultaneously
// each of them will modify value in RAM and wait for update to be flushed to disk.
// As soon as data of last update was flushed to disk - all updates consider
// operation as being successful.
//
// This technique allows us to execute 1000s of sequential updates to a set of
// releated db records under single ID, without having to wait for each update
// to be flushed to disk.
package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
)

type Store struct {
	db      *pebble.DB
	kmu     []*kmutex
	mu      sync.Mutex
	done    chan struct{}
	count   int  // number of requests processed from last WAL write
	stopped bool // graceful shudown
	pending int  // number of requests inflight (track for graceful shutdown)
}

// Having multiple mutexes reduces on sync.Cond and sync.Mutex
// proportional to amount of mutexes.
// This consumes just a few kb of memory, but provides significant
// boost to performance
const mCount = 100

func NewStore(db *pebble.DB) *Store {
	s := &Store{
		db:   db,
		done: make(chan struct{}),
	}
	for i := 0; i < mCount; i++ {
		s.kmu = append(s.kmu, newLocker())
	}
	return s
}

// Flush ensure that all in-memory writes that happened before had
// been flushed to persistent storage.
// In this code writes are written as "async" pebble writes, which
// means pebble manages timing on it's own. What we do here is just
// issues single Sync write to WAL and wait for it to complete, ensuring that
// all async writes before were flushed to WAL
func (p *Store) Flush() int {
	p.mu.Lock()
	count := p.count
	p.count = 0
	done := p.done // all previous updates are waiting on this chan
	pending := p.pending
	p.done = make(chan struct{}) // create new chan for future updates to wait on
	p.mu.Unlock()

	if count > 0 {
		err := p.db.LogData([]byte("f"), pebble.Sync)
		if err != nil {
			panic(err)
		}
	}
	close(done)
	return pending
}

// FlushLoop calls Flush constantly in a loop
func (p *Store) FlushLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			p.mu.Lock()
			p.stopped = true // make sure all new requests are failing
			p.mu.Unlock()
			for {
				pending := p.Flush() // flush all pending requests
				if pending == 0 {
					return nil
				}
			}
		default:
			n := p.Flush()
			if n == 0 {
				// avoid infinite loops if no data needs to be flushed
				time.Sleep(time.Millisecond * 1)
			}
		}
	}
}

// UpdateFunc should update only data relevant to the key.
// It can create multiple records in DB, but they should
// never overlap with data of another keys
type UpdateFunc func() error

// UpdateFunc should update only data relevant to the key.
// It can create multiple records in DB, but they should
// never overlap with data of another keys
type GetFunc func(db *pebble.DB) error

// singletonUpdate makes sure all updates are done one after the other.
func (p *Store) singletonUpdate(key []byte, f UpdateFunc) error {
	if len(key) > 0 {
		h := fnv.New64a()
		h.Write(key)
		kid := h.Sum64()
		p.kmu[kid%mCount].Lock(kid)
		defer p.kmu[kid%mCount].Unlock(kid)
	}
	return f()
}

// Update the data for the key using UpdateFunc.
// UpdateFunc will simply call Store
func (p *Store) Update(key []byte, f UpdateFunc) error {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return fmt.Errorf("DB stopped")
	}
	p.pending++
	p.count++
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.pending--
		p.mu.Unlock()
	}()

	err := p.singletonUpdate(key, f)
	if err != nil {
		return err
	}

	// wait till our update is flushed to disk
	p.mu.Lock()
	done := p.done
	p.mu.Unlock()
	<-done
	return nil
}

// copied this implementation from someone on the web
type kmutex struct {
	c *sync.Cond
	l sync.Locker
	s map[uint64]struct{}
}

func newLocker() *kmutex {
	l := sync.Mutex{}
	return &kmutex{c: sync.NewCond(&l), l: &l, s: make(map[uint64]struct{})}
}

func (km *kmutex) locked(key uint64) (ok bool) {
	_, ok = km.s[key]
	return
}

func (km *kmutex) Unlock(key uint64) {
	km.l.Lock()
	defer km.l.Unlock()
	delete(km.s, key)
	km.c.Broadcast()
}

func (km *kmutex) Lock(key uint64) {
	km.l.Lock()
	defer km.l.Unlock()
	for km.locked(key) {
		km.c.Wait()
	}
	km.s[key] = struct{}{}
}
