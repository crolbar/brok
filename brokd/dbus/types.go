package dbus

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

type MsgType byte

const (
	MSG_INVALID MsgType = 0 + iota
	MSG_METHOD_CALL
	MSG_METHOD_RETURN
	MSG_ERROR
	MSG_SIGNAL
)

type MsgFlag byte

const (
	FLAG_NO_REPLY_EXPECTED MsgFlag = 1 << iota
	FLAG_NO_AUTO_START
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
	Str string
}

type Variant struct {
	Sig   Signature
	Value any
}

type header struct {
	Field HeaderField
	Variant
}

type Msg struct {
	Type MsgType

	Headers map[HeaderField]Variant
	Body    []byte
}

type Call struct {
	serial int
	flags  byte
	method string
	path   string
	dest   string
	body   []byte
}
