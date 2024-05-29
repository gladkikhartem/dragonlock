package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v2"

	"github.com/buaazp/fasthttprouter"
	_ "github.com/mattn/go-sqlite3"

	_ "github.com/lib/pq"
)

type Config struct {
	ListenAddr   string `yaml:"ListenAddr"`
	DBPath       string `yaml:"DBPath"`
	SnapshotPath string `yaml:"SnapshotPath"`
	BackupPath   string `yaml:"BackupPath"`
}

func main() {
	// f, err := os.Create("prof.txt")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = pprof.StartCPUProfile(f)
	// if err != nil {
	// 	panic(err)
	// }
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// defer func() {
	// 	pprof.StopCPUProfile()
	// 	log.Print("STPO")
	// 	os.Exit(1)
	// }()

	// go func() {
	// 	http.ListenAndServe(":8082", nil)
	// }()
	err := Start(ctx)
	if err != nil {
		panic(err)
	}
}

var store *Store

func Start(ctx context.Context) error {
	cfgFilename := os.Getenv("CFG_FILE")
	if cfgFilename == "" {
		cfgFilename = "config.yml"
	}
	yd, err := os.ReadFile(cfgFilename)
	if err != nil {
		return err
	}
	var cfg Config
	err = yaml.Unmarshal(yd, &cfg)
	if err != nil {
		return err
	}

	opts := &pebble.Options{}
	opts.Levels = []pebble.LevelOptions{
		{
			Compression:    pebble.ZstdCompression,
			TargetFileSize: 1024 * 1024 * 50,
		},
		{
			Compression:    pebble.ZstdCompression,
			TargetFileSize: 1024 * 1024 * 100,
		},
		{
			Compression:    pebble.ZstdCompression,
			TargetFileSize: 1024 * 1024 * 200,
		},
		{
			Compression:    pebble.ZstdCompression,
			TargetFileSize: 1024 * 1024 * 500,
		},
	}
	db, err := pebble.Open(cfg.DBPath, opts)
	if err != nil {
		return err
	}

	store = NewStore(db)

	go func() {
		log.Print("START ", cfg.ListenAddr)
		router := fasthttprouter.New()
		// In memory APIs. Not persistent
		router.POST("/db/:acc/flock/:id", FastLockHandler)
		router.DELETE("/db/:acc/flock/:id", FastUnlockHandler)

		// Persistent APIs
		router.GET("/db/:acc/sequence/:id", GetSequenceHandler)
		router.POST("/db/:acc/sequence/:id", NextSequenceHandler)
		router.DELETE("/db/:acc/sequence/:id", DeleteSequenceHandler)

		router.GET("/db/:acc/counter/:id", GetCounterHandler)
		router.POST("/db/:acc/counter/:id", AddCounterHandler)
		router.DELETE("/db/:acc/counter/:id", DeleteCounterHandler)

		router.GET("/db/:acc/kv/:id", GetKVHandler)
		router.POST("/db/:acc/kv/:id", SetKVHandler)
		router.DELETE("/db/:acc/kv/:id", DeleteKVHandler)

		router.GET("/db/:acc/lock/:id", GetLockHandler)
		router.POST("/db/:acc/lock/:id", SetLockHandler)
		router.DELETE("/db/:acc/lock/:id", DeleteLockHandler)

		router.NotFound = func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(404)
		}

		s := fasthttp.Server{
			Handler:       router.Handler,
			Concurrency:   100000,
			MaxConnsPerIP: 100000,
		}
		err := s.ListenAndServe(cfg.ListenAddr)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		err = store.FlushLoop(ctx)
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Second * 1)
	log.Print("Running BenchmarkFastLocks1000Clients1000RandomKeys")
	start := time.Now()
	BenchmarkFastLocks1000Clients1000RandomKeys()
	log.Print("1m requests took sec: ", time.Since(start).Seconds())
	time.Sleep(time.Second * 1)
	start = time.Now()
	BenchmarkSequence1000Clients1Key()
	log.Print("1m requests took sec: ", time.Since(start).Seconds())
	time.Sleep(time.Second * 1)

	return nil
}

func BenchmarkSequence1000Clients1Key() {
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

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
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
	if string(rreesp.Body()) != fmt.Sprint(100000) {
		panic("COUNTER MISMATCH " + string(rreesp.Body()) + " " + fmt.Sprint(100000))
	}

	fasthttp.ReleaseURI(counterURI)
}

func BenchmarkFastLocks1000Clients1000RandomKeys() {
	c := fasthttp.Client{}
	c.MaxConnsPerHost = 10000
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				lockURI := fasthttp.AcquireURI()
				lockURI.Parse(nil, []byte(fmt.Sprintf("http://localhost:8081/db/123/flock/%d?dur=10&wait=10", j)))
				unlockURI := fasthttp.AcquireURI()
				unlockURI.Parse(nil, []byte(fmt.Sprintf("http://localhost:8081/db/123/flock/%d", j)))

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
