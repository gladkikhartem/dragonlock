package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"

	"net/http"
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
	f, err := os.Create("prof.txt")
	if err != nil {
		log.Fatal(err)
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		panic(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	defer func() {
		pprof.StopCPUProfile()
		log.Print("STPO")
		os.Exit(1)
	}()

	go func() {
		http.ListenAndServe(":8082", nil)
	}()
	err = Start(ctx)
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
			MaxConnsPerIP: 1000000,
		}
		err := s.ListenAndServe(cfg.ListenAddr)
		if err != nil {
			panic(err)
		}
	}()

	err = store.FlushLoop(ctx)
	if err != nil {
		panic(err)
	}

	return nil
}
