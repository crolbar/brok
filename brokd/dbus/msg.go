package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
)

func writeHeader(buf *bytes.Buffer, h header) {
	pad(buf, 8)

	// FIELD
	buf.WriteByte(byte(h.Field))

	// SIG
	buf.WriteByte(byte(len(h.sig.str))) // len of sig
	b := make([]byte, len(h.sig.str)+1)
	copy(b, h.sig.str)
	b[len(b)-1] = 0 // null terminate sig
	buf.Write(b)

	// VALUE
	switch v := h.value.(type) {
	case string:
		binary.Write(buf, binary.LittleEndian, uint32(len(v)))
		b := make([]byte, len(v)+1)
		copy(b, v)
		b[len(b)-1] = 0
		buf.Write(b)
	default:
		panic("header value type not supported")
	}
}

func (d *Dbus) call(method string, path string, dest string, body ...string) {
	iface := ""
	i := strings.LastIndex(method, ".")
	if i != -1 {
		iface = method[:i]
	}
	method = method[i+1:]

	var (
		b  bytes.Buffer
		bw = func(data any) {
			binary.Write(&b, binary.LittleEndian, data)
		}

		headers = []header{
			{Field: FieldPath, Variant: Variant{value: path, sig: Signature{str: "o"}}},
			{Field: FieldDestination, Variant: Variant{value: dest, sig: Signature{str: "s"}}},

			{Field: FieldMember, Variant: Variant{value: method, sig: Signature{str: "s"}}},
			{Field: FieldInterface, Variant: Variant{value: iface, sig: Signature{str: "s"}}},
		}

		// we need the len of the headers in bytes, this is the easiest way
		headersBuf bytes.Buffer

		serial = d.getSerial()
	)

	b.WriteByte('l')      // little edian
	b.WriteByte(1)        // type: method call
	b.WriteByte(0)        // no flags
	b.WriteByte(1)        // protocol version
	bw(uint32(len(body))) // body size
	bw(uint32(serial))

	for _, header := range headers {
		writeHeader(&headersBuf, header)
	}

	bw(uint32(headersBuf.Len()))
	pad(&b, 8)
	b.Write(headersBuf.Bytes())
	pad(&b, 8)

	d.writeCh <- b.Bytes()
}

func readHeaders(buf []byte, order binary.ByteOrder, headers map[HeaderField]Variant) {
	if len(buf) == 0 {
		return
	}
	// TODO: assuming we are at padding
	if buf[0] == 0 {
		return
	}

	var (
		h header
		i = 0
	)

	h.Field = HeaderField(buf[i])
	i += 1

	sigLen := buf[i]
	i += 1

	h.sig = Signature{str: string(buf[i : i+int(sigLen)])}
	i += int(sigLen) + // sig
		1 // null term

	switch h.Field {

	// uint32
	case FieldReplySerial, FieldUnixFDs:
		var v uint32
		binary.Read(bytes.NewBuffer(buf[i:]), order, &v)
		h.value = v
		i += 4

	// string
	case FieldSender,
		FieldInterface,
		FieldMember,
		FieldErrorName,
		FieldDestination,
		FieldPath:
		var length uint32
		binary.Read(bytes.NewBuffer(buf[i:]), order, &length)
		i += 4

		var v = make([]byte, length)
		binary.Read(bytes.NewBuffer(buf[i:]), order, &v)
		i += int(length) + 1
		h.value = string(v)

		// skip the padding
		n := 4 + int(length) + 1
		if n%4 != 0 {
			i += 4 - (n % 4)
		}

	case FieldSignature:
		length := buf[i]
		i += 1

		v := buf[i : i+int(length)]
		h.value = v
		i += int(length) + 1
	}

	headers[h.Field] = h.Variant
	readHeaders(buf[i:], order, headers)
}

func (d *Dbus) readMsg(fixedHeader [16]byte) Msg {
	var (
		order binary.ByteOrder

		bodySize   uint32
		headerSize uint32

		headers map[HeaderField]Variant = make(map[HeaderField]Variant)
	)

	switch fixedHeader[0] {
	case 'l':
		order = binary.LittleEndian
	case 'B':
		order = binary.BigEndian
	default:
		panic("dbus invalid msg read from server")
	}

	binary.Read(bytes.NewBuffer(fixedHeader[4:8]), order, &bodySize)
	binary.Read(bytes.NewBuffer(fixedHeader[12:]), order, &headerSize)

	// header size field does not include the padding at the end, add it
	if headerSize%8 != 0 {
		headerSize += 8 - (headerSize % 8)
	}

	headersBuf := make([]byte, headerSize)
	_, err := io.ReadFull(d.C, headersBuf)
	if err != nil {
		panic(err)
	}
	readHeaders(headersBuf, order, headers)

	body := make([]byte, bodySize)
	_, err = io.ReadFull(d.C, body)
	if err != nil {
		panic(err)
	}

	return Msg{
		Type:    fixedHeader[1],
		headers: headers,
		body:    body,
	}
}
