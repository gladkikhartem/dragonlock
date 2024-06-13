package cd

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *KV) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Data":
			z.Data, err = dc.ReadBytes(z.Data)
			if err != nil {
				err = msgp.WrapError(err, "Data")
				return
			}
		case "Version":
			z.Version, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "Version")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *KV) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Data"
	err = en.Append(0x82, 0xa4, 0x44, 0x61, 0x74, 0x61)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Data)
	if err != nil {
		err = msgp.WrapError(err, "Data")
		return
	}
	// write "Version"
	err = en.Append(0xa7, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Version)
	if err != nil {
		err = msgp.WrapError(err, "Version")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *KV) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Data"
	o = append(o, 0x82, 0xa4, 0x44, 0x61, 0x74, 0x61)
	o = msgp.AppendBytes(o, z.Data)
	// string "Version"
	o = append(o, 0xa7, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o = msgp.AppendInt64(o, z.Version)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *KV) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Data":
			z.Data, bts, err = msgp.ReadBytesBytes(bts, z.Data)
			if err != nil {
				err = msgp.WrapError(err, "Data")
				return
			}
		case "Version":
			z.Version, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Version")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *KV) Msgsize() (s int) {
	s = 1 + 5 + msgp.BytesPrefixSize + len(z.Data) + 8 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Lock) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "o":
			z.Handle, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "Handle")
				return
			}
		case "t":
			z.Till, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "Till")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Lock) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "o"
	err = en.Append(0x82, 0xa1, 0x6f)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Handle)
	if err != nil {
		err = msgp.WrapError(err, "Handle")
		return
	}
	// write "t"
	err = en.Append(0xa1, 0x74)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Till)
	if err != nil {
		err = msgp.WrapError(err, "Till")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Lock) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "o"
	o = append(o, 0x82, 0xa1, 0x6f)
	o = msgp.AppendInt64(o, z.Handle)
	// string "t"
	o = append(o, 0xa1, 0x74)
	o = msgp.AppendInt64(o, z.Till)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Lock) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "o":
			z.Handle, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Handle")
				return
			}
		case "t":
			z.Till, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Till")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Lock) Msgsize() (s int) {
	s = 1 + 2 + msgp.Int64Size + 2 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *QueueMeta) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Total":
			z.Total, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "Total")
				return
			}
		case "Counter":
			z.Counter, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "Counter")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z QueueMeta) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Total"
	err = en.Append(0x82, 0xa5, 0x54, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Total)
	if err != nil {
		err = msgp.WrapError(err, "Total")
		return
	}
	// write "Counter"
	err = en.Append(0xa7, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Counter)
	if err != nil {
		err = msgp.WrapError(err, "Counter")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z QueueMeta) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Total"
	o = append(o, 0x82, 0xa5, 0x54, 0x6f, 0x74, 0x61, 0x6c)
	o = msgp.AppendInt64(o, z.Total)
	// string "Counter"
	o = append(o, 0xa7, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72)
	o = msgp.AppendInt64(o, z.Counter)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *QueueMeta) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Total":
			z.Total, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Total")
				return
			}
		case "Counter":
			z.Counter, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Counter")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z QueueMeta) Msgsize() (s int) {
	s = 1 + 6 + msgp.Int64Size + 8 + msgp.Int64Size
	return
}
