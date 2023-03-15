package wsclient

type DataFrame struct {
	Op    int         `json:"op"`   //opcode (shows what datatype)
	Data  interface{} `json:"data"` //contains data
	Event string      `json:"event"`
}

type helloFrame struct {
	HeartbeatInterval int `json:"heartbeatInterval"`
}

type helloResFrame struct {
	Token string `json:"token"`
}
