package cd

//go:generate msgp
type Lock struct {
	Handle int64 `json:"o" msg:"o"`
	Till   int64 `json:"t" msg:"t"`
}

//go:generate msgp
type KV struct {
	Data []byte `json:"d" msg:"d"`
}
