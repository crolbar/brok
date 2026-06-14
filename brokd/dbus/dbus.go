package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

var (
	dest = "org.freedesktop.DBus"
	path = "/org/freedesktop/DBus"
)

type Dbus struct {
	C *net.UnixConn

	serial   int
	lastBody []byte
}

type HeaderField byte

const (
	FieldPath HeaderField = 1 + iota
	FieldInterface
	FieldMember
	FieldErrorName
	FieldReplySerial
	FieldDestination
	FieldSender
	FieldSignature
	FieldUnixFDs
	fieldMax
)

var headerMap = map[HeaderField]string{
	0: "INVALID",
	1: "PATH",
	2: "INTERFACE",
	3: "MEMBER",
	4: "ERROR_NAME",
	5: "REPLY_SERIAL",
	6: "DESTINATION",
	7: "SENDER",
	8: "SIGNATURE",
	9: "UNIX_FDS",
}

type Signature struct {
	str string
}

type Variant struct {
	sig   Signature
	value any
}

type header struct {
	Field byte
	Variant
}

type Msg struct {
	Type byte

	headers []header
	body    []byte
}

func (m Msg) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Msg: %d\n", m.Type))
	sb.WriteString("headers:\n")
	for _, h := range m.headers {
		sb.WriteString("  [")
		switch h.Field {
		case byte(FieldPath):
			sb.WriteString("FieldPath")
		case byte(FieldInterface):
			sb.WriteString("FieldInterface")
		case byte(FieldMember):
			sb.WriteString("FieldMember")
		case byte(FieldErrorName):
			sb.WriteString("FieldErrorName")
		case byte(FieldReplySerial):
			sb.WriteString("FieldReplySerial")
		case byte(FieldDestination):
			sb.WriteString("FieldDestination")
		case byte(FieldSender):
			sb.WriteString("FieldSender")
		case byte(FieldSignature):
			sb.WriteString("FieldSignature")
		case byte(FieldUnixFDs):
			sb.WriteString("FieldUnixFDs")
		}
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprintf("%q", h.value))
		sb.WriteString(fmt.Sprintf("%s", h.sig))
		sb.WriteString("]\n")
	}

	sb.WriteString(fmt.Sprintf("body: %s", string(m.body)))

	return sb.String()
}

func pad(b *bytes.Buffer, align int) {
	for b.Len()%align != 0 {
		b.WriteByte(0)
	}
}

func writeHeader(buf *bytes.Buffer, h header) {
	pad(buf, 8)

	// FIELD
	buf.WriteByte(h.Field)

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
			{Field: byte(FieldPath), Variant: Variant{value: path, sig: Signature{str: "o"}}},
			{Field: byte(FieldDestination), Variant: Variant{value: dest, sig: Signature{str: "s"}}},

			{Field: byte(FieldMember), Variant: Variant{value: method, sig: Signature{str: "s"}}},
			{Field: byte(FieldInterface), Variant: Variant{value: iface, sig: Signature{str: "s"}}},
		}

		// we need the len of the headers in bytes, this is the easiest way
		headersBuf bytes.Buffer
	)

	b.WriteByte('l')      // little edian
	b.WriteByte(1)        // type: method call
	b.WriteByte(0)        // no flags
	b.WriteByte(1)        // protocol version
	bw(uint32(len(body))) // body size
	bw(uint32(d.serial))
	d.serial += 1

	for _, header := range headers {
		writeHeader(&headersBuf, header)
	}

	bw(uint32(headersBuf.Len()))
	pad(&b, 8)
	b.Write(headersBuf.Bytes())
	pad(&b, 8)

	_, err := b.WriteTo(d.C)
	if err != nil {
		panic(err)
	}
}

func readHeaders(buf []byte, order binary.ByteOrder, headers *[]header) {
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

	h.Field = buf[i]
	i += 1

	sigLen := buf[i]
	i += 1

	h.sig = Signature{str: string(buf[i : i+int(sigLen)])}
	i += int(sigLen) + // sig
		1 // null term

	switch h.Field {

	// uint32
	case byte(FieldReplySerial), byte(FieldUnixFDs):
		var v uint32
		binary.Read(bytes.NewBuffer(buf[i:]), order, &v)
		h.value = v
		i += 4

	// string
	case byte(FieldSender),
		byte(FieldInterface),
		byte(FieldMember),
		byte(FieldErrorName),
		byte(FieldDestination),
		byte(FieldPath):
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

	case byte(FieldSignature):
		length := buf[i]
		i += 1

		v := buf[i : i+int(length)]
		h.value = v
		i += int(length) + 1
	}

	*headers = append(*headers, h)
	readHeaders(buf[i:], order, headers)
}

func (d *Dbus) readMsg(fixedHeader [16]byte) Msg {
	var (
		order binary.ByteOrder

		bodySize   uint32
		headerSize uint32

		headers []header
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
	readHeaders(headersBuf, order, &headers)

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

func (d *Dbus) reader() {
	for {
		var fixedHeader [16]byte
		_, err := io.ReadFull(d.C, fixedHeader[:])
		if err != nil {
			panic(err)
		}
		msg := d.readMsg(fixedHeader)
		fmt.Println(msg)
	}
}

func (d *Dbus) test() {
	fmt.Println(string(d.lastBody))
}

func main() {
	var (
		address = os.Getenv("DBUS_SESSION_BUS_ADDRESS")

		splitAddr = strings.Split(strings.Split(address, ";")[0], "=")
		t         = splitAddr[0]
		path      = splitAddr[1]
	)

	if t != "unix:path" {
		panic("dubs addr in DBUS_SESSION_BUS_ADDRESS not supported")
	}

	C, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		panic(err)
	}

	dbus := Dbus{C: C, serial: 1}

	err = dbus.Auth()
	if err != nil {
		panic(err)
	}
	fmt.Println("connected")

	go dbus.reader()

	dbus.call("org.freedesktop.DBus.Hello", path, dest)
	// dbus.test()
	// dbus.call("org.freedesktop.DBus.Peer.Ping")
	// dbus.call("org.freedesktop.DBus.GetId")

	select {}
}
