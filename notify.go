package main

import (
	"sync"
	"time"
)

func newNotifier() *notifier {
	l := sync.Mutex{}
	return &notifier{c: sync.NewCond(&l), l: &l, s: make(map[string]*NotifierRecord)}
}

type NotifierRecord struct {
	Version   int64
	Listeners int
}

type notifier struct {
	c *sync.Cond
	l sync.Locker
	s map[string]*NotifierRecord
}

func (km *notifier) NotifyVersion(key string, ver int64) {
	km.l.Lock()
	defer km.l.Unlock()
	v, ok := km.s[key]
	if !ok { // no listeners - no need to notify
		return
	}
	v.Version = ver
	km.c.Broadcast()
}

// make sure that NotifierRecord won't be cleared between time we check version in db
// and the time we start listening for changes
func (km *notifier) Attach(key string, ver int64) {
	km.l.Lock()
	defer km.l.Unlock()

	v, ok := km.s[key]
	if !ok {
		v = &NotifierRecord{
			Listeners: 1,
		}
		km.s[key] = v
		return
	}
	v.Listeners++
}

func (km *notifier) Listen(key string, ver int64, dur int) int64 {
	start := time.Now().Unix()
	km.l.Lock()
	defer km.l.Unlock()

	for {
		v, ok := km.s[key]
		if ok && v.Version != 0 && v.Version != ver { // changed!
			v.Listeners--
			return v.Version
		}

		if start+int64(dur) < time.Now().Unix() { // timeout
			v.Listeners--
			if v.Listeners == 0 {
				delete(km.s, key) // no one listening - free up RAM
			}
			return -1
		}
		km.c.Wait()
	}
}
