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
		switch msg.Type {
		case MSG_METHOD_RETURN, MSG_ERROR:
			if _, ok := msg.headers[FieldReplySerial]; ok {
				replySerial := int(msg.headers[FieldReplySerial].value.(uint32))

				d.replyChMu.Lock()
				replyCh := d.replyChs[replySerial]
				d.replyChMu.Unlock()

				replyCh <- msg
			}
		case MSG_SIGNAL:

			// mpris update (sig sa{sv}as)
			if msg.headers[FieldPath].value.(string) == "/org/mpris/MediaPlayer2" {
				// fmt.Println(msg)
				props := ParsePropertiesChanged(msg.body)

				sender := msg.headers[FieldSender].value.(string)
				fmt.Println(sender, props)
			}
			// client created/destoryed
			if msg.headers[FieldPath].value.(string) == "/org/freedesktop/DBus" &&
				msg.headers[FieldMember].value.(string) == "NameOwnerChanged" {
				// fmt.Println(msg)
			}
		}

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

	msg = dbus.Call("org.freedesktop.DBus.ListNames")
	a := ParseStringArray(msg.body)
	for _, e := range a {
		if strings.HasPrefix(e, "org.mpris.") {
			fmt.Println(e)
		}
	}
	msg = dbus.Call("org.freedesktop.DBus.GetNameOwner", WithBody([]byte("org.mpris.MediaPlayer2.spotify")))
	fmt.Println(msg)

	dbus.Call("org.freedesktop.DBus.AddMatch",
		WithBody([]byte("type='signal',interface='org.freedesktop.DBus.Properties'")),
	)
	dbus.Call("org.freedesktop.DBus.AddMatch",
		WithBody([]byte("type='signal',interface='org.freedesktop.DBus',member='NameOwnerChanged'")),
	)

	// msg = dbus.Call("org.mpris.MediaPlayer2.Player.PlayPause",
	// 	WithPath("/org/mpris/MediaPlayer2"),
	// 	// WithDest("org.mpris.MediaPlayer2.mpd"))
	// WithDest("org.mpris.MediaPlayer2.vivaldi"))
	// fmt.Println(msg)

	fmt.Println("end")
	select {}
}
