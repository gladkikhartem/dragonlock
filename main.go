package main

import (
	"context"
	"log"
	"os"
	"os/signal"

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
	err := Start()
	if err != nil {
		panic(err)
	}
}

var store *Store

func Start() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

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
		router.GET("/fastlock/:id", FastLockHandler)
		router.POST("/fastunlock/:id", FastUnlockHandler)

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

		err := fasthttp.ListenAndServe(cfg.ListenAddr, router.Handler)
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
