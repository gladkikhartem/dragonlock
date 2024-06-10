package main

import (
	"clouddragon/cd"

	"github.com/valyala/fasthttp"
)

// Add new message(s) to Queue
func EnqueueHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	err = store.Update(compID(cd.FIFOPrefix, acc, id), func() error {
		// ADD TO QUEUE
		return nil
	})
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
}

// Remove old message(s) from Queue and fetch new one(s) for processing
func DequeueAndAckHandler(ctx *fasthttp.RequestCtx) {
	acc, id, err := getAccID(ctx)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
	err = store.Update(compID(cd.FIFOPrefix, acc, id), func() error {
		// POP FROM QUEUE
		return nil
	})
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}
}
