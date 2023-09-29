package events

import (
	"regexp"
	"time"
)

type UnreadMsg struct {
	MsgId    int64     `json:"msgId,string"` //message last read Id
	Count    int       `json:"count"`        //number of unread messages
	Time     time.Time `json:"time"`
	Mentions int       `json:"mentions"` //how many times user was mentioned
}

type Msg struct { //id and request id not omitted for checking purposes
	MsgId            int64         `json:"id,string"`
	GuildId          int64         `json:"guildId,string"` // Chat id
	RequestId        string        `json:"requestId"`
	Content          string        `json:"content"` // message content
	MentionsEveryone *bool         `json:"mentionsEveryone,omitempty"`
	Mentions         *[]User       `json:"mentions,omitempty"`
	Attachments      *[]Attachment `json:"attachments,omitempty"`
	Author           User          `json:"author,omitempty"` // author id aka user id
	Created          time.Time     `json:"created,omitempty"`
	Modified         time.Time     `json:"modified,omitempty"`
	MsgSaved         bool          `json:"msgSaved,omitempty"` //shows if the message is saved or not
}

type Attachment struct {
	Id int64 `json:"id,string"`
	//ContentType string `json:"contentType"` //file type
	Filename string `json:"filename"`
	Type     string `json:"type"`
}

var MentionExp = regexp.MustCompile(`\<\@(\d+)\>`)
var MentionEveryoneExp = regexp.MustCompile(`\<\@everyone\>`)
