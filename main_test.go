package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func BenchmarkSequence1000Clients1Key(b *testing.B) {
	var wg sync.WaitGroup
	counterURI := fasthttp.AcquireURI()
	counterURI.Parse(nil, []byte("http://localhost:8081/db/123/sequence/1"))

	c := fasthttp.Client{}
	c.MaxConnsPerHost = 10000
	rreq := fasthttp.AcquireRequest()
	rreesp := fasthttp.AcquireResponse()
	rreq.SetURI(counterURI)
	rreq.Header.SetMethod("DELETE")
	err := c.Do(rreq, rreesp)
	if err != nil {
		panic(err)
	}

	counter := int64(0)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				if atomic.AddInt64(&counter, 1) > int64(b.N) {
					return
				}
				req := fasthttp.AcquireRequest()
				req.Header.SetMethod("POST")
				req.SetURI(counterURI)
				resp := fasthttp.AcquireResponse()
				err := c.Do(req, resp)
				if err != nil {
					panic(err)
				}
				if resp.StatusCode() != 200 {
					panic(resp.StatusCode())
				}
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
			}
		}(i)
	}
	wg.Wait()

	rreq.Header.SetMethod("GET")
	err = c.Do(rreq, rreesp)
	if err != nil {
		panic(err)
	}
	if string(rreesp.Body()) != fmt.Sprint(b.N) {
		panic("COUNTER MISMATCH " + string(rreesp.Body()) + " " + fmt.Sprint(b.N))
	}

	fasthttp.ReleaseURI(counterURI)
}

func BenchmarkFastLocks1000Clients1000RandomKeys(b *testing.B) {
	c := fasthttp.Client{}
	c.MaxConnsPerHost = 10000
	var wg sync.WaitGroup

	counter := int64(0)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if atomic.AddInt64(&counter, 1) > int64(b.N) {
					return
				}
				rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
				id := rnd.Intn(1000)
				lockURI := fasthttp.AcquireURI()
				lockURI.Parse(nil, []byte(fmt.Sprintf("http://localhost:8081/db/123/flock/%d?dur=10&wait=10", id)))
				unlockURI := fasthttp.AcquireURI()
				unlockURI.Parse(nil, []byte(fmt.Sprintf("http://localhost:8081/db/123/flock/%d", id)))

				req := fasthttp.AcquireRequest()
				req.Header.SetMethod("POST")
				resp := fasthttp.AcquireResponse()

				req.SetURI(lockURI)
				err := c.Do(req, resp)
				if err != nil {
					panic(err)
				}
				if resp.StatusCode() != 200 {
					panic(string(resp.Body()))
				}
				req.SetURI(unlockURI)
				req.Header.SetMethod("DELETE")
				err = c.Do(req, resp)
				if err != nil {
					panic(err)
				}
				if resp.StatusCode() != 200 {
					panic(string(resp.Body()))
				}
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
				fasthttp.ReleaseURI(lockURI)
				fasthttp.ReleaseURI(unlockURI)
			}
		}()
	}
	wg.Wait()
}
