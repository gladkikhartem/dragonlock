package cd

import "errors"

const (
	SequentialIDPrefix = 1
	CounterPrefix      = 2
	LocksPrefix        = 3
	KVPrefix           = 4
	FIFOPrefix         = 5
)

var ErrNotLocked = errors.New("not_locked")

//go:generate msgp
type Lock struct {
	Handle int64 `json:"o" msg:"o"`
	Till   int64 `json:"t" msg:"t"`
}

//go:generate msgp
type KV struct {
	Data []byte `json:"d" msg:"d"`
}
