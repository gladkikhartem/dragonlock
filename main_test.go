package main

import (
	"context"
	"os"
	"os/signal"
	"testing"

	"github.com/buaazp/fasthttprouter"
	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

func TestMain(m *testing.M) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	os.RemoveAll("test.db")
	opts := &pebble.Options{}
	db, err := pebble.Open("test.db", opts)
	if err != nil {
		panic(err)
	}

	store = NewStore(db)

	go func() {
		router := fasthttprouter.New()
		router.GET("/db/:acc/sequence/:id", GetSequenceHandler)
		router.POST("/db/:acc/sequence/:id", NextSequenceHandler)
		router.DELETE("/db/:acc/sequence/:id", DeleteSequenceHandler)
		router.GET("/db/:acc/counter/:id", GetCounterHandler)
		router.POST("/db/:acc/counter/:id", AddCounterHandler)
		router.DELETE("/db/:acc/counter/:id", DeleteCounterHandler)
		router.GET("/db/:acc/kv/:id", GetKVHandler)
		router.POST("/db/:acc/kv/:id", SetKVHandler)
		router.DELETE("/db/:acc/kv/:id", DeleteKVHandler)
		err := fasthttp.ListenAndServe(":33123", router.Handler)
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

	os.Exit(m.Run())
}
