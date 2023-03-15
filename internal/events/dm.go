package events

type Dm struct {
	DmId      int64     `json:"id"`
	UserInfo  User      `json:"userInfo"`
	UnreadMsg UnreadMsg `json:"unreadMsg"`
}
