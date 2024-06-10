package main

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/cockroachdb/pebble"
	"github.com/valyala/fasthttp"
)

type Getter interface {
	Get(key []byte) ([]byte, io.Closer, error)
}
type Setter interface {
	Set(key, value []byte, _ *pebble.WriteOptions) error
}

func Int64ToByte(val int64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(val))
	return buf
}
func ByteToInt64(d []byte) int64 {
	return int64(binary.LittleEndian.Uint64(d))
}

func GetInt64(key []byte, b Getter) (*int64, error) {
	d, closer, err := b.Get([]byte(key))
	if err != nil && err != pebble.ErrNotFound {
		return nil, fmt.Errorf("DB ERR %v", err.Error())
	}
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	defer closer.Close()
	seq := int64(binary.LittleEndian.Uint64(d))
	return &seq, nil
}

func SetInt64(key []byte, val int64, b Setter) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(val))
	return b.Set(key, buf, pebble.NoSync)
}

// TableID|Account|0|ID
// 0 byte delimited is used to construct composite key from Acc and ID
func compID(prefix int, acc, id string) []byte {
	b := make([]byte, 0, len(id)+len(acc)+2)
	b = append(b, byte(prefix))
	b = append(b, acc...)
	b = append(b, 0)
	b = append(b, id...)
	return b
}

// TableID|ID
// 0 byte delimited is used to construct composite key from Acc and ID
func compID1(prefix int, id string) []byte {
	b := make([]byte, 0, len(id)+1)
	b = append(b, byte(prefix))
	b = append(b, id...)
	return b
}

// TableID|ID
// 0 byte delimited is used to construct composite key from Acc and ID
func fromCompID1(key []byte) string {
	return string(key[1:])
}

func getAccID(ctx *fasthttp.RequestCtx) (string, string, error) {
	acc := ctx.UserValue("acc").(string)
	if len(acc) > 255 || len(acc) == 0 {
		return "", "", fmt.Errorf("len is not in range 0~255")
	}
	for _, v := range acc {
		if v == 0 {
			return "", "", fmt.Errorf("0 is not allowed as a character in acc name")
		}
	}
	id := ctx.UserValue("id").(string)
	if len(id) > 255 || len(id) == 0 {
		return "", "", fmt.Errorf("id is not in range 0~255")
	}
	for _, v := range id {
		if v == 0 {
			return "", "", fmt.Errorf("0 is not allowed as a character in id")
		}
	}
	return acc, id, nil
}
