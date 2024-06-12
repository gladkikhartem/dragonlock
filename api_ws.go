package main

import (
	"github.com/valyala/fasthttp"
)

// This API combines
// - Service discovery (list of active WS connections),
// - State management (all WS connections receive info about version increments)
// - Leader election (all nodes are sorted consistently, which means 1st node can be chosen a leader)
// - Queue management (receiving message from queues)
//
// This works pretty good for service discovery and dynamic state management and
// background workers doing their work
//
// Slaves will push data to queue, that is processed by one node and than this node
// can update the state, which in turn sent out immediately to all other nodes.
//
// This can provide a framework for distributed systems management and serve as generic
// queue mechanism

func GetStateHandler(ctx *fasthttp.RequestCtx) {
	// open websocket, receive freshest value, get notified about newest values
	// AKA pub/sub, but with delivery guarantees
	// (i.e. in case TCP fails - we will reconnect and ensure we have latest
	// version of state
}

func SetStateHandler(ctx *fasthttp.RequestCtx) {
	// update state and make sure all websockets for this key receive an update
}
