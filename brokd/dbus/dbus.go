package dbus

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
)

type Dbus struct {
	C    *net.UnixConn
	Name string

	writeCh chan []byte

	replyChMu sync.Mutex
	replyChs  map[int](chan Msg)

	serial int

	signalCh *chan Msg
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
			if d.signalCh != nil {
				*d.signalCh <- msg
			}
		}
	}
}

func NewSession() (*Dbus, error) {
	var (
		address = os.Getenv("DBUS_SESSION_BUS_ADDRESS")

		splitAddr = strings.Split(strings.Split(address, ";")[0], "=")
		t         = splitAddr[0]
		path      = splitAddr[1]
	)

	if t != "unix:path" {
		return nil, errors.New("dubs addr in DBUS_SESSION_BUS_ADDRESS not supported")
	}

	C, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, err
	}

	dbus := Dbus{
		C: C,

		writeCh: make(chan []byte),
		serial:  0,

		replyChs: make(map[int]chan Msg),

		signalCh: nil,
	}

	err = dbus.Auth()
	if err != nil {
		return nil, err
	}

	go dbus.reader()
	go dbus.writer()

	msg := dbus.Call("org.freedesktop.DBus.Hello")
	if len(msg.body) < 6 {
		return nil, errors.New("len of reply msg body for Hello is invalid")
	}
	dbus.Name = string(msg.body[4 : len(msg.body)-1])

	return &dbus, nil
}
