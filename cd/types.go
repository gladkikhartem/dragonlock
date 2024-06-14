package cd

import "errors"

const (
	AtomicPrefix      = 1 // storage for atomic counters
	VerSequencePrefix = 3 // store increasing version numbers for KV
	LocksPrefix       = 4 // store lock durations to restore in case of reboot
	IdempotencyPrefix = 5 // store idempotency keys to deduplicate requests
	KVPrefix          = 6 // store kv values
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
