package events

type Dm struct {
	DmId     int64     `json:"id,string"`
	UserInfo User      `json:"userInfo"`
	Unread   UnreadMsg `json:"unread"`
}
