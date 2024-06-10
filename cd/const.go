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
