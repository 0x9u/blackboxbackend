package wsclient

import "time"

const (
	pingDelay = 20 * time.Second
	//messageLimit = 2 << 7 //256
	pongDelay    = (pingDelay * 2) / 5
	tokenTimeout = 15 * time.Second
)

const (
	TYPE_DISPATCH = iota //enum
	TYPE_IDENTIFY
	TYPE_HELLO
	TYPE_CLOSE        = 0x8 //(used when client has done something invalid causing ws to close)
	TYPE_HEARTBEAT    = 0x9
	TYPE_HEARTBEATACK = 0xa
)
