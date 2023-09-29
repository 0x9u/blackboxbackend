package events

import (
	"regexp"
	"time"

	"github.com/asianchinaboi/backendserver/internal/errors"
)

// used for guild settings, info, join info, update info
type Guild struct {
	OwnerId  int64      `json:"ownerId,omitempty,string"`
	Dm       *bool      `json:"dm,omitempty"`
	GuildId  int64      `json:"id,string"`
	Name     string     `json:"name,omitempty"`
	ImageId  int64      `json:"imageId,omitempty,string"`
	Unread   *UnreadMsg `json:"unread,omitempty"`
	SaveChat *bool      `json:"saveChat,omitempty"`
}

type Invite struct {
	Invite  string `json:"invite"`
	GuildId int64  `json:"guildId,string"`
}

type Typing struct { //basically member but without perms and with a dm param
	GuildId  int64     `json:"guildId,string"`
	UserInfo User      `json:"userInfo"`
	Time     time.Time `json:"time"` //used for client to sync with other clients in case of latency issues
	//if 5 seconds have passed since timestamp client will discard
}

func ValidateGuildInput(body *Guild) (errors.ErrCode, error) {

	nameValid, err := ValidateGuildName(body.Name)
	if err != nil {
		return errors.StatusInternalError, err
	}
	if !nameValid {
		return errors.StatusInvalidGuildName, errors.ErrInvalidGuildName
	}
	return -1, nil
}

func ValidateGuildName(name string) (bool, error) {
	return regexp.MatchString(`^[\x20-\xFF]{6,64}$`, name)
}
