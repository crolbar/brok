package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
)

type Dbus struct {
	C *net.UnixConn

	writeCh chan []byte

	replyChMu sync.Mutex
	replyChs  map[int](chan Msg)

	serial   int
	lastBody []byte
}

func (d *Dbus) writer() {
	for {
		b := <-d.writeCh

		_, err := d.C.Write(b)
		if err != nil {
			panic(fmt.Sprintln("write err:", err))
		}
	}
}

func (d *Dbus) reader() {
	for {
		var fixedHeader [16]byte
		_, err := io.ReadFull(d.C, fixedHeader[:])
		if err != nil {
			panic(fmt.Sprintln("read err:", err))
		}
		msg := d.readMsg(fixedHeader)

		// send the msg back to the reply chan
		if _, ok := msg.headers[FieldReplySerial]; ok {
			replySerial := int(msg.headers[FieldReplySerial].value.(uint32))

			d.replyChMu.Lock()
			replyCh := d.replyChs[replySerial]
			d.replyChMu.Unlock()

			replyCh <- msg
		}

		// fmt.Println(msg)
		// fmt.Println()
	}
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
		serial:  0,

		replyChs: make(map[int]chan Msg),
	}

	err = dbus.Auth()
	if err != nil {
		panic(err)
	}
	fmt.Println("connected")

	go dbus.reader()
	go dbus.writer()

	msg := dbus.Call("org.freedesktop.DBus.Hello")
	fmt.Println("name", string(msg.body))

	msg = dbus.Call("org.freedesktop.DBus.Peer.Ping")
	fmt.Println(msg)

	msg = dbus.Call("org.freedesktop.DBus.GetId")
	fmt.Println(msg)

	// msg = dbus.Call("org.mpris.MediaPlayer2.Player.PlayPause",
	// 	WithPath("/org/mpris/MediaPlayer2"),
	// 	WithDest("org.mpris.MediaPlayer2.vivaldi.instance4043"))
	// fmt.Println(msg)

	fmt.Println("end")
	select {}
}
