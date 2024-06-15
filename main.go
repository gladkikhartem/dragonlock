package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	_ "net/http/pprof"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
	"gopkg.in/yaml.v2"

	"github.com/buaazp/fasthttprouter"
	_ "github.com/mattn/go-sqlite3"

	_ "github.com/lib/pq"
)

type Config struct {
	ListenAddr string         `yaml:"ListenAddr"`
	DBPath     string         `yaml:"DBPath"`
	DBOptions  pebble.Options `yaml:"DBOptions"`
	// TODO: backups & restore from S3
	//
	// S3 speed:  ~1GB/s per avg instance   6GB/sec network-optimized
	// 1GB -> ~10 sec backup
	// 10GB -> ~10 sec bacvkup
	// 100GB -> 1.5 min backup   <- ideal DB size
	// 1TB -> 15 min <- too big :(
	// 10TB -> 2.7 hours <- tooooo big :()
	//
	// Can use "S3 sync" to reduce amount of data updated & not having to merge
	// sstables every time
	// if SSTable is not changed - only diffs  & WAL will be uploaded
	//
	// Can make periodic backups though (daily, monthly)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err := Start(ctx)
	if err != nil {
		panic(err)
	}
}

var store *Store

func Start(ctx context.Context) error {
	var cfg Config
	yd, err := os.ReadFile("config.yml")
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yd, &cfg)
	if err != nil {
		return err
	}
	db, err := pebble.Open(cfg.DBPath, &cfg.DBOptions)
	if err != nil {
		return err
	}
	store = NewStore(db)
	InitFastLocks()
	go func() {
		log.Print("START ", cfg.ListenAddr)
		router := fasthttprouter.New()
		router.POST("/req/:acc", RequestHandler)
		router.POST("/watch/:acc", WatchHandler)

		router.NotFound = func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(404)
		}

		s := fasthttp.Server{
			Handler:                       router.Handler,
			Concurrency:                   100000,
			MaxConnsPerIP:                 100000,
			ReadBufferSize:                10000,
			WriteBufferSize:               10000,
			DisableHeaderNamesNormalizing: true,
			NoDefaultContentType:          true,
			NoDefaultDate:                 true,
			NoDefaultServerHeader:         true,
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
