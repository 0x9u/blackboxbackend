package events

type Dm struct {
	DmId      int       `json:"id"`
	UserInfo  User      `json:"userInfo"`
	UnreadMsg UnreadMsg `json:"unreadMsg"`
}
