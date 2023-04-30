package events

import (
	"regexp"
	"time"
)

type UnreadMsg struct {
	Id       int64     `json:"msgId,string"` //message last read Id
	Count    int       `json:"count"`        //number of unread messages
	Time     time.Time `json:"time"`
	Mentions int       `json:"mentions"` //how many times user was mentioned
}

type Msg struct { //id and request id not omitted for checking purposes
	MsgId            int64        `json:"id,string"`
	Author           User         `json:"author,omitempty,string"`  // author id aka user id
	Content          string       `json:"content"`                  // message content
	GuildId          int64        `json:"guildId,omitempty,string"` // Chat id
	Created          time.Time    `json:"created,omitempty"`
	Modified         time.Time    `json:"modified,omitempty"`
	MsgSaved         bool         `json:"msgSaved,omitempty"` //shows if the message is saved or not
	RequestId        string       `json:"requestId"`
	MentionsEveryone bool         `json:"mentionsEveryone"`
	Mentions         []User       `json:"mentions"`
	Attachments      []Attachment `json:"attachments"`
}

type Attachment struct {
	Id int64 `json:"id,string"`
	//ContentType string `json:"contentType"` //file type
	Filename string `json:"filename"`
}

var MentionExp = regexp.MustCompile(`\<\@(\d+)\>`)
var MentionEveryoneExp = regexp.MustCompile(`\@everyone`)
