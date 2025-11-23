package main

import (
	"net"

	"github.com/crolbar/brok/share"
)

type M struct {
	conn net.Conn
}

func (m *M) sendMsg(msg string) {
	if len(msg) > share.MAX_MSG_LEN {
		panic("msg len")
	}

	_, err := m.conn.Write([]byte("msg:"+msg))
	if err != nil {
		panic(err)
	}
}

func main() {
	conn, err := net.Dial("unix", "/tmp/brokd.sock")
	if err != nil {
		panic(err)
	}

	m := M{conn: conn}

	m.sendMsg(share.MSG_NEXT)

	// buf := make([]byte, 20)
	// n, err := conn.Read(buf)
	// if err != nil {
	// 	panic(err)
	// }

	// println(string(buf[:n]))
}
