package cd

import "errors"

const (
	AtomicPrefix      = 1
	VerSequencePrefix = 2
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
	Data    []byte
	Version int64
}

type QueueMeta struct {
	Total   int64 // total messages in a queue
	Counter int64 // id of the last message
}
