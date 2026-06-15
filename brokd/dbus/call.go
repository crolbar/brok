package dbus

import (
	"bytes"
	"encoding/binary"
	"strings"
)

func writeHeader(buf *bytes.Buffer, h header) {
	pad(buf, 8)

	// FIELD
	buf.WriteByte(byte(h.Field))

	// SIG
	buf.WriteByte(byte(len(h.Sig.Str))) // len of sig
	b := make([]byte, len(h.Sig.Str)+1)
	copy(b, h.Sig.Str)
	b[len(b)-1] = 0 // null terminate sig
	buf.Write(b)

	// VALUE
	switch h.Field {
	case FieldSender,
		FieldInterface,
		FieldMember,
		FieldErrorName,
		FieldDestination,
		FieldPath:
		v := h.Value.(string)
		binary.Write(buf, binary.LittleEndian, uint32(len(v)))
		b := make([]byte, len(v)+1)
		copy(b, v)
		b[len(b)-1] = 0
		buf.Write(b)

	case FieldSignature:
		v := h.Value.(string)
		if len(v) > 254 {
			panic("len of sig long")
		}
		buf.WriteByte(byte(len(v)))
		b := make([]byte, len(v)+1)
		copy(b, v)
		b[len(b)-1] = 0
		buf.Write(b)

	case FieldReplySerial, FieldUnixFDs:
		fallthrough
	default:
		panic("header value type not supported")
	}
}

// TODO: BODY SUPPORTS ONLY SINGLE STRING
func (d *Dbus) makeCall(c Call) Msg {
	iface := ""
	i := strings.LastIndex(c.method, ".")
	if i != -1 {
		iface = c.method[:i]
	}
	c.method = c.method[i+1:]

	var (
		b  bytes.Buffer
		bw = func(data any) {
			binary.Write(&b, binary.LittleEndian, data)
		}

		headers = []header{
			{Field: FieldPath, Variant: Variant{Value: c.path, Sig: Signature{Str: "o"}}},
			{Field: FieldDestination, Variant: Variant{Value: c.dest, Sig: Signature{Str: "s"}}},

			{Field: FieldMember, Variant: Variant{Value: c.method, Sig: Signature{Str: "s"}}},
			{Field: FieldInterface, Variant: Variant{Value: iface, Sig: Signature{Str: "s"}}},
		}

		// we need the len of the headers in bytes, this is the easiest way
		headersBuf bytes.Buffer

		reply chan Msg = make(chan Msg)

		bodySize = 0
	)
	if len(c.body) > 0 {
		bodySize = len(c.body) + 4 + 1

		headers = append(headers, header{
			Field: FieldSignature,
			// TODO: fixed value just for strings, change if needed for other types in body
			Variant: Variant{Value: "s", Sig: Signature{Str: "g"}},
		})
	}

	{
		for _, header := range headers {
			writeHeader(&headersBuf, header)
		}
	}

	b.WriteByte('l')             // little edian
	b.WriteByte(1)               // type: method call
	b.WriteByte(c.flags)         // flags
	b.WriteByte(1)               // protocol version
	bw(uint32(bodySize))         // body size
	bw(uint32(c.serial))         // serial
	bw(uint32(headersBuf.Len())) // headers len
	b.Write(headersBuf.Bytes())  // headers
	pad(&b, 8)

	// body
	if len(c.body) > 0 {
		bw(uint32(len(c.body)))
		b.Write(c.body)
		b.WriteByte(0)
	}

	if c.flags&byte(FLAG_NO_REPLY_EXPECTED) == 0 {
		d.replyChMu.Lock()
		d.replyChs[c.serial] = reply
		d.replyChMu.Unlock()

		d.writeCh <- b.Bytes()

		return <-reply
	}
	return Msg{}
}

func (d *Dbus) Call(method string, opts ...CallOpt) Msg {
	call := Call{
		flags:  0,
		serial: d.getSerial(),
		dest:   "org.freedesktop.DBus",
		path:   "/org/freedesktop/DBus",
		method: method,
	}

	for _, opt := range opts {
		opt(&call)
	}

	return d.makeCall(call)
}
