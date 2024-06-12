package cd

import "errors"

const (
	AtomicPrefix      = 1
	LocksPrefix       = 3
	KVPrefix          = 4
	FIFOPrefix        = 5
	IdempotencyPrefix = 6
	QueueMetaPrefix   = 7
	QueueMsgPrefix    = 8
)

var ErrNotLocked = errors.New("not_locked")

//go:generate msgp
type Lock struct {
	Handle int64 `json:"o" msg:"o"`
	Till   int64 `json:"t" msg:"t"`
}

//go:generate msgp
type KV struct {
	Data string `json:"d" msg:"d"`
}

type QueueMeta struct {
	Front int64 // Front points to the first message in the queue.
	Back  int64 // Back points to the last message in the queue
	// If queue is empty: Front = Back + 1
}
