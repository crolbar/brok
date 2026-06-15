package main

import (
	"bytes"
	"fmt"
	"strings"
)

func (d *Dbus) getSerial() int {
	d.serial += 1
	return d.serial
}

func (m Msg) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Msg: %d\n", m.Type))
	sb.WriteString("headers:\n")
	for k, v := range m.headers {
		sb.WriteString("  [")
		switch k {
		case FieldPath:
			sb.WriteString("FieldPath")
		case FieldInterface:
			sb.WriteString("FieldInterface")
		case FieldMember:
			sb.WriteString("FieldMember")
		case FieldErrorName:
			sb.WriteString("FieldErrorName")
		case FieldReplySerial:
			sb.WriteString("FieldReplySerial")
		case FieldDestination:
			sb.WriteString("FieldDestination")
		case FieldSender:
			sb.WriteString("FieldSender")
		case FieldSignature:
			sb.WriteString("FieldSignature")
		case FieldUnixFDs:
			sb.WriteString("FieldUnixFDs")
		}
		sb.WriteString(", ")
		sb.WriteString(fmt.Sprintf("%q", v.value))
		sb.WriteString(fmt.Sprintf("%s", v.sig))
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
