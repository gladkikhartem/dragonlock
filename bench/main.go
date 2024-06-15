package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

func BenchmarkAtomic(u string, accs, keys, parallel, nPerThread int) {
	c := fasthttp.Client{
		MaxConnsPerHost: 50000,
	}
	uris := make([]*fasthttp.URI, accs)
	for i := 0; i < accs; i++ {
		uris[i] = fasthttp.AcquireURI()
		err := uris[i].Parse(nil, []byte(u+fmt.Sprintf("/req/%d", i)))
		if err != nil {
			panic(err)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		b1 := []byte(`{"Atomic": [{"Key": "`)
		b2 := []byte(`", "Add": 1}]}`)
		wg.Add(1)
		go func() {
			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			defer wg.Done()
			for j := 0; j < nPerThread; j++ {
				b := append(b1, []byte(fmt.Sprint(rnd.Intn(keys)))...)
				b = append(b, b2...)
				req := fasthttp.AcquireRequest()
				req.Header.SetMethod("POST")
				req.SetBody(b)
				req.SetURI(uris[rnd.Intn(accs)])
				resp := fasthttp.AcquireResponse()
				err := c.Do(req, resp)
				if err != nil {
					panic(err)
				}
				if resp.StatusCode() != 200 {
					panic(fmt.Sprintf("NON 200 sstatus code: %v %v ", resp.StatusCode(), string(resp.Body())))
				}
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkKV(u string, accs, keys, parallel, nPerThread, size int) {
	c := fasthttp.Client{
		MaxConnsPerHost: 50000,
	}
	uris := make([]*fasthttp.URI, accs)
	for i := 0; i < accs; i++ {
		uris[i] = fasthttp.AcquireURI()
		err := uris[i].Parse(nil, []byte(u+fmt.Sprintf("/req/%d", i)))
		if err != nil {
			panic(err)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		b1 := []byte(`{"KVSet": [{"Key": "`)
		b2 := []byte(`", "Value": "`)
		b3 := []byte(`"}]}`)
		wg.Add(1)
		go func() {
			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			defer wg.Done()
			for j := 0; j < nPerThread; j++ {
				b := append(b1, []byte(fmt.Sprint(rnd.Intn(keys)))...)
				b = append(b, b2...)
				for s := 0; s < size; s++ {
					b = append(b, byte(rand.Intn(26)+'A'))
				}
				b = append(b, b3...)
				// log.Print(string(b))
				req := fasthttp.AcquireRequest()
				req.Header.SetMethod("POST")
				req.SetBody(b)
				req.SetURI(uris[rnd.Intn(accs)])
				resp := fasthttp.AcquireResponse()
				err := c.Do(req, resp)
				if err != nil {
					panic(err)
				}
				if resp.StatusCode() != 200 {
					panic(fmt.Sprintf("NON 200 sstatus code: %v %v ", resp.StatusCode(), string(resp.Body())))
				}
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkLockUnlock(u string, accs, keys, parallel, nPerThread int) {
	c := fasthttp.Client{
		MaxConnsPerHost: 50000,
	}
	uris := make([]*fasthttp.URI, accs)
	for i := 0; i < accs; i++ {
		uris[i] = fasthttp.AcquireURI()
		err := uris[i].Parse(nil, []byte(u+fmt.Sprintf("/req/%d", i)))
		if err != nil {
			panic(err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		b1 := []byte(`{"LockDur": 30, "LockWait": 30, "LockID": "`)
		b2 := []byte(`"}`)
		u1 := []byte(`{"UnlockID": "`)
		u2 := []byte(`"}`)

		wg.Add(1)
		go func() {
			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			defer wg.Done()
			for j := 0; j < nPerThread; j++ {
				ur := uris[rnd.Intn(accs)]
				key := []byte(fmt.Sprint(rnd.Intn(keys)))
				b := append(b1, key...)
				b = append(b, b2...)
				req := fasthttp.AcquireRequest()
				req.Header.SetMethod("POST")
				req.SetBody(b)
				req.SetURI(ur)
				resp := fasthttp.AcquireResponse()
				err := c.Do(req, resp)
				if err != nil {
					panic(err)
				}
				if resp.StatusCode() != 200 {
					panic(fmt.Sprintf("NON 200 sstatus code: %v %v ", resp.StatusCode(), string(resp.Body())))
				}

				ub := append(u1, key...)
				ub = append(ub, u2...)

				req.Header.SetMethod("POST")
				req.SetBody(ub)
				req.SetURI(ur)
				err = c.Do(req, resp)
				if err != nil {
					panic(err)
				}
				if resp.StatusCode() != 200 {
					panic(fmt.Sprintf("NON 200 sstatus code: %v %v ", resp.StatusCode(), string(resp.Body())))
				}

				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkWatchReaction(u string, parallel int) time.Duration {
	c := fasthttp.Client{
		MaxConnsPerHost: 50000,
	}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := rnd.Int63()
	uri := fasthttp.AcquireURI()
	err := uri.Parse(nil, []byte(u+"/watch/1"))
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		b := []byte(fmt.Sprintf(`{"ID": "%d", "Version": 0}`, id))
		wg.Add(1)
		go func() {
			defer wg.Done()
			// log.Print(string(b))
			req := fasthttp.AcquireRequest()
			req.Header.SetMethod("POST")
			req.SetBody(b)
			req.SetURI(uri)
			resp := fasthttp.AcquireResponse()
			err := c.Do(req, resp)
			if err != nil {
				panic(err)
			}
			if resp.StatusCode() != 200 {
				panic(fmt.Sprintf("NON 200 sstatus code: %v %v ", resp.StatusCode(), string(resp.Body())))
			}
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
		}()
	}
	time.Sleep(time.Millisecond * 300)
	kvuri := fasthttp.AcquireURI()
	err = kvuri.Parse(nil, []byte(u+"/req/1"))
	if err != nil {
		panic(err)
	}

	start := time.Now()
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("POST")
	req.SetBody([]byte(fmt.Sprintf(`{"KVSet": [{"Key": "%v", "Value": "1"}]}`, id)))
	req.SetURI(kvuri)
	resp := fasthttp.AcquireResponse()
	err = c.Do(req, resp)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != 200 {
		panic(fmt.Sprintf("NON 200 sstatus code: %v %v ", resp.StatusCode(), string(resp.Body())))
	}
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	wg.Wait()
	return time.Since(start)
}

func main() {
	parallel := 100
	perThread := 1
	total := float64(parallel * perThread)
	start := time.Now()

	took := BenchmarkWatchReaction("http://localhost:8081", 1)
	log.Printf("WatchReaction 1 key 1 watcher: %d ms ", took.Milliseconds())

	took = BenchmarkWatchReaction("http://localhost:8081", 100)
	log.Printf("WatchReaction 1 key 100 watchers: %d ms ", took.Milliseconds())

	took = BenchmarkWatchReaction("http://localhost:8081", 1000)
	log.Printf("WatchReaction 1 key 100 watchers: %d ms ", took.Milliseconds())

	took = BenchmarkWatchReaction("http://localhost:8081", 1000)
	log.Printf("WatchReaction 1 key 10000 watchers: %d ms ", took.Milliseconds())

	var wg sync.WaitGroup
	var mu sync.Mutex
	tt := []int64{}
	start = time.Now()
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			took = BenchmarkWatchReaction("http://localhost:8081", 1)
			tk := took.Milliseconds()
			mu.Lock()
			tt = append(tt, tk)
			mu.Unlock()
		}()
	}
	wg.Wait()
	sum := int64(0)
	max := int64(0)
	min := int64(999999)
	for _, t := range tt {
		sum += t
		if max < t {
			max = t
		}
		if min > t {
			min = t
		}
	}
	log.Printf("WatchReaction 1000 keys 5 watchers per key: min %d ms,avg %.1f ms,  max %d ms,  total delay: %d ms ",
		min, float64(sum)/float64(len(tt)), max, time.Since(start).Milliseconds())

	parallel = 1000
	perThread = 100
	total = float64(parallel * perThread)

	start = time.Now()
	BenchmarkLockUnlock("http://localhost:8081", 1, 1000, parallel, perThread)
	log.Printf("LockUnlock for 1 account and 1000 keys: %.1fk req/sec ", total/time.Since(start).Seconds()/1000)

	start = time.Now()
	BenchmarkLockUnlock("http://localhost:8081", 1000, 1000, parallel, perThread)
	log.Printf("LockUnlock for 1000 accounts and 1000 keys: %.1fk req/sec ", total/time.Since(start).Seconds()/1000)

	size := 64
	BenchmarkKV("http://localhost:8081", 1, 1, parallel, perThread, size)
	log.Printf("64byte write KV for 1 account and 1 key: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1, 1000, parallel, perThread, size)
	log.Printf("64byte write KV for 1 account and 1000 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1000, 1, parallel, perThread, size)
	log.Printf("64byte write KV for 1000 accounts and 1 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1000, 1000, parallel, perThread, size)
	log.Printf("64byte write KV for 1000 accounts and 1000 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())

	size = 1024
	BenchmarkKV("http://localhost:8081", 1, 1, parallel, perThread, size)
	log.Printf("1KB write KV for 1 account and 1 key: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1, 1000, parallel, perThread, size)
	log.Printf("1KB write KV for 1 account and 1000 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1000, 1, parallel, perThread, size)
	log.Printf("1KB write KV for 1000 accounts and 1 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1000, 1000, parallel, perThread, size)
	log.Printf("1KB write KV for 1000 accounts and 1000 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())

	perThread = 10
	total = float64(parallel * perThread)
	size = 10240
	BenchmarkKV("http://localhost:8081", 1, 1, parallel, perThread, size)
	log.Printf("10 KB write KV for 1 account and 1 key: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1, 1000, parallel, perThread, size)
	log.Printf("10 KB write KV for 1 account and 1000 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1000, 1, parallel, perThread, size)
	log.Printf("10 KB write KV for 1000 accounts and 1 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())
	start = time.Now()
	BenchmarkKV("http://localhost:8081", 1000, 1000, parallel, perThread, size)
	log.Printf("10 KB write KV for 1000 accounts and 1000 keys: %.1fk req/sec %.1f MB/sec", total/time.Since(start).Seconds()/1000, total*float64(size)/1000000/time.Since(start).Seconds())

	start = time.Now()
	BenchmarkAtomic("http://localhost:8081", 1, 1, parallel, perThread)
	log.Printf("atomic for 1 account and 1 key: %.1fk req/sec", total/time.Since(start).Seconds()/1000)
	start = time.Now()
	BenchmarkAtomic("http://localhost:8081", 1, 1000, parallel, perThread)
	log.Printf("atomic for 1 account and 1000 keys: %.1fk req/sec", total/time.Since(start).Seconds()/1000)
	start = time.Now()
	BenchmarkAtomic("http://localhost:8081", 1000, 1, parallel, perThread)
	log.Printf("atomic for 1000 accounts and 1 keys: %.1fk req/sec", total/time.Since(start).Seconds()/1000)
	start = time.Now()
	BenchmarkAtomic("http://localhost:8081", 1000, 1000, parallel, perThread)
	log.Printf("atomic for 1000 accounts and 1000 keys: %.1fk req/sec", total/time.Since(start).Seconds()/1000)
}
