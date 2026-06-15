package main

import (
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

	writeCh chan []byte
	serial   int
	lastBody []byte
}

func (d *Dbus) writer() {
	for {
		b := <-d.writeCh

		_, err := d.C.Write(b)
		if err != nil {
			panic(err)
		}
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

	dbus := Dbus{
		C: C,

		writeCh: make(chan []byte),
		serial:  1,
	}

	err = dbus.Auth()
	if err != nil {
		panic(err)
	}
	fmt.Println("connected")

	go dbus.reader()
	go dbus.writer()

	dbus.call("org.freedesktop.DBus.Hello", path, dest)
	// dbus.test()
	// dbus.call("org.freedesktop.DBus.Peer.Ping")
	// dbus.call("org.freedesktop.DBus.GetId")

	select {}
}
