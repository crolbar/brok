package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"github.com/crolbar/brok/share"
)

type M struct {
	conn net.Conn
}

func (m *M) sendMsg(msg string) {
	if len(msg) > share.MAX_MSG_LEN-4 {
		panic("msg len")
	}

	_, err := m.conn.Write([]byte("msg:" + msg))
	if err != nil {
		panic(err)
	}
}

func (m *M) listener() {
	for {
		headerBuf := make([]byte, 2)
		n, err := m.conn.Read(headerBuf)
		if err != nil {
			panic(err)
		}
		if n != 2 {
			panic("no 2 byte header")
		}

		size := binary.LittleEndian.Uint16(headerBuf)

		buf := make([]byte, size)
		n, err = m.conn.Read(buf)
		if err != nil {
			panic(err)
		}

		if n != int(size) {
			panic("incorrect size of body send from server")
		}

		data := buf[:n]

		fmt.Println(string(data))
	}
}

func main() {
	conn, err := net.Dial("unix", share.SockPath)
	if err != nil {
		panic(err)
	}

	m := M{conn: conn}

	noArgs := true
	for _, arg := range os.Args {
		switch arg {
		case "next", "--next":
			m.sendMsg(share.MSG_NEXT)
		case "prev", "--prev", "previous", "--previous":
			m.sendMsg(share.MSG_PREV)
		case "play-pause", "--play-pause":
			m.sendMsg(share.MSG_PLAY_PAUSE)

		case "sub", "subscribe", "--subscribe":
			m.sendMsg(share.MSG_SUB)
			m.listener()

		case "quit":
			m.sendMsg("quit")
		default:
			continue
		}
		noArgs = false
	}

	if noArgs {
		fmt.Println("\x1b[31mNo arg provided\x1b[m")
	}
}
