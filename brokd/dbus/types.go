package main

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
	Field HeaderField
	Variant
}

type Msg struct {
	Type byte

	headers map[HeaderField]Variant
	body    []byte
}
