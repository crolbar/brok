package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	serial int = 1

	dest = "org.freedesktop.DBus"
	path = "/org/freedesktop/DBus"
)

func authReadLine(C net.Conn) ([][]byte, error) {
	in := bufio.NewReader(C)

	msgBuf, err := in.ReadBytes('\n')
	if err != nil {
		return [][]byte{}, err
	}

	bytes.TrimSuffix(msgBuf, []byte("\r\n"))
	return bytes.Split(msgBuf, []byte{' '}), nil
}

func authWriteLine(C net.Conn, cmds ...[]byte) error {
	buf := make([]byte, 0)

	for i, c := range cmds {
		buf = append(buf, c...)
		if i != len(cmds)-1 {
			buf = append(buf, ' ')
		}
	}

	buf = append(buf, '\r')
	buf = append(buf, '\n')
	n, err := C.Write(buf)
	if n != len(buf) {
		panic("write err n != len(buf)")
	}
	return err
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

func call(C net.Conn, method string, body ...string) {
	var (
		b  bytes.Buffer
		bw = func(data any) {
			binary.Write(&b, binary.LittleEndian, data)
		}

		headers = []header{
			{Field: byte(FieldPath), Variant: Variant{value: path, sig: Signature{str: "o"}}},
			{Field: byte(FieldDestination), Variant: Variant{value: dest, sig: Signature{str: "s"}}},

			{Field: byte(FieldMember), Variant: Variant{value: method, sig: Signature{str: "s"}}},
		}
	)

	b.WriteByte('l')      // little edian
	b.WriteByte(1)        // type: method call
	b.WriteByte(0)        // no flags
	b.WriteByte(1)        // protocol version
	bw(uint32(len(body))) // body size
	bw(uint32(serial))
	serial += 1

	_, err := b.WriteTo(C)
	if err != nil {
		panic(err)
	}
	bb := make([]byte, 2)
	fmt.Println("reading")
	C.Read(bb)
	fmt.Println(bb)
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

	{
		// ucred := &syscall.Ucred{Pid: int32(os.Getpid()), Uid: uint32(os.Getuid()), Gid: uint32(os.Getgid())}
		// b := syscall.UnixCredentials(ucred)
		_, _, err := C.WriteMsgUnix([]byte{0}, []byte{}, nil)
		if err != nil {
			panic(err)
		}
	}

	{
		err := authWriteLine(C, []byte("AUTH"))
		if err != nil {
			panic(err)
		}
		cmds, err := authReadLine(C)
		if err != nil {
			panic(err)
		}
		if string(cmds[0]) != "REJECTED" && string(cmds[1]) != "EXTERNAL" {
			panic("auth protocol error: expected REJECTED AND EXTERNAL")
		}
	}

	{
		uid := strconv.Itoa(os.Getuid())
		b := make([]byte, 2*len(uid))
		hex.Encode(b, []byte(uid))

		authWriteLine(C, []byte("AUTH EXTERNAL"), b)

		cmds, err := authReadLine(C)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(cmds[0], []byte("OK")) {
			panic("dbus auth: status not OK")
		}
	}

	// NOTE: NOT USED
	// uuid := cmds[1]

	err = authWriteLine(C, []byte("NEGOTIATE_UNIX_FD"))
	if err != nil {
		panic(err)
	}

	cmds, err := authReadLine(C)
	if bytes.Equal(cmds[0], []byte("ERROR")) {
		panic("dbus auth: NEGOTION UNIX FD FAILED")
	}

	err = authWriteLine(C, []byte("BEGIN"))
	if err != nil {
		panic(err)
	}

	fmt.Println("connected")

	call(C, "org.freedesktop.DBus.Hello")

	// fmt.Println("red: ", string(msgBuf))

	select {}
}
