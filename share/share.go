package share

const (
	MAX_MSG_LEN = 100

	SockPath = "/tmp/brokd.sock"

	MSG_NEXT = "next"
	MSG_PREV = "prev"
	MSG_PLAY_PAUSE = "playpause"
	MSG_SUB = "sub"

	MSG_FOCUS = "focus"
	MSG_FOCUS_LEN = len(MSG_FOCUS)
)
