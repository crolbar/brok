package dbus

import (
	"bytes"
	"encoding/binary"
	"io"
)

func readHeaders(buf []byte, order binary.ByteOrder, headers map[HeaderField]Variant) {
	if len(buf) == 0 {
		return
	}
	if buf[0] == 0 {
		panic("at 0")
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
		if i%8 != 0 {
			i += 8 - (i % 8)
		}

	case FieldSignature:
		length := buf[i]
		i += 1

		v := buf[i : i+int(length)]
		h.value = v
		i += int(length) + 1

		if i%8 != 0 {
			i += 8 - (i % 8)
		}
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
		Type:    MsgType(fixedHeader[1]),
		headers: headers,
		body:    body,
	}
}
