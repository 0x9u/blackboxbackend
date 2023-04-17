package events

type Dm struct {
	DmId     int64     `json:"id"`
	UserInfo User      `json:"userInfo"`
	Unread   UnreadMsg `json:"unread"`
}
