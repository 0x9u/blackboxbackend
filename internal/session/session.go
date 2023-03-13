package session

type Session struct {
	Expires int64        `json:"expires"`
	Id      int64        `json:"-"`
	Token   string       `json:"token,omitempty"`
	Perms   *Permissions `json:"perms,omitempty"`
}

type Permissions struct { //admin stuff
	Guilds Self `json:"guild"`
	Users  Self `json:"user"`
	BanIP  bool `json:"banIP"`
	Admin  bool `json:"admin"` //all perms basically (only admin can reset database)
}

type Self struct {
	Delete bool `json:"delete"`
	Get    bool `json:"get"`
	Edit   bool `json:"edit"`
}
