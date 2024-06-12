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

/*

// UPDATE - execute writes synchronously for 1 DB
//
// LOCK/UNLOCK - work on individual lock logic
//
*/

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

		// TODO: make API unified, so that one can issue N
		// writes to DB and be sure that all of them complete as a
		// single batch (atomicity guarantees, aka transactions,
		// but for API.
		//
		// probably requires some unified JSON API or something like that
		// similar to ElasticSearch batch operations
		//
		// Intentionally don't support multiple chained updates.
		// Every operation should be finished via atomic write to
		// multiple places
		//
		// Another aspect is idempotency - we probably want
		// to store  last XXX idempotency keys in DB to ensure
		// that when user locks DB - he can check if operation was already
		// performed or failed in processing

		// UNIFIED API:
		// ALL OPERATIONS ARE EXECUTED ATOMICALLY AFTER SUCCESSFUL
		// LOCK / EXTEND operations (during UNLOCK - extend called, then write performed, then unlock called)
		//
		// ALL OPERATIONS ARE EXECUTED ATOMICALLY.
		// ERRORS MAKE ALL WRITES FAIL, so need to be very careful with what
		// can cause error and what cannot.
		//
		// ALL BATCHES CAN BE WRITTEN with IdempotencyID(s).
		// Subsequent requests
		// with this IdempotencyID will return DUPLICATE error.
		// Multiple IDs should be allowed to make sure that if batch is retried
		// with different set of messages - it will still give an error with info
		// about which IDs are in conflict (useful for batching multiple writes together)
		/*
			POST /db/123
			{
				"lock": true,

				"dequeue": {
					"ckf": 10,
				},
				"ack": [1231232,12312312,123123213],
				"inc": {
					"ctr": 1,
					"abc": -1
				}
				"seq": {
					"wtf": 1
				}
				"kv_get": ["123","345"],
			}
			POST /db/123
			{
				"extend_lock": "handleid",

				"enqueue":  {
					"ckf": [.....],
				}
				"dequeue": {
					"ckf": 10,
				},
				"ack": [1231232,12312312,123123213],
				"get": ["123","345"],
				"set": {
					"123": "val",
					"345": null
				}
				"del": ["1233","3333"],
			}
			POST /db/123
			{
				"unlock": "handleid",

				"enqueue": "abc",
				"dequeue": "cdf",
				"ack": "eff",

				"get": ["123","345"],
				"set": {
					"123": "val",
					"345": null
				}
				"del": ["1233","3333"],
			}
		*/

		// TODO: unify locking mechanism:
		// internal locks don't have timeout goroutine & don't have lock durations
		// lock is applied to batch, that is sent to store
		//
		// Everything is wrapped by  Update(), but depending on operations list
		// we may want (or don't want) to write locks to DB and create goroutines.
		//
		// If explicit locking is performed - we will have to make sure lock
		// is flushed together with operations result to DB. (in a single batch)
		// and we need to schedule automatic unlock
		// //

		router.POST("/db/:acc", RequestHandler)

		// router.POST("/db/:acc/lock/:id", FastLockHandler)
		// router.DELETE("/db/:acc/lock/:id", FastUnlockHandler)

		// router.GET("/db/:acc/seq/:id", GetSequenceHandler)
		// router.POST("/db/:acc/seq/:id", NextSequenceHandler)
		// router.DELETE("/db/:acc/seq/:id", DeleteSequenceHandler)

		// router.GET("/db/:acc/cnt/:id", GetCounterHandler)
		// router.POST("/db/:acc/cnt/:id", AddCounterHandler)
		// router.DELETE("/db/:acc/cnt/:id", DeleteCounterHandler)

		// router.GET("/db/:acc/kv/:id", GetKVHandler)
		// router.POST("/db/:acc/kv/:id", SetKVHandler)
		// router.DELETE("/db/:acc/kv/:id", DeleteKVHandler)

		// router.POST("/db/:acc/fifo/:id", EnqueueHandler)
		// router.DELETE("/db/:acc/fifo/:id", DequeueAndAckHandler)

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
