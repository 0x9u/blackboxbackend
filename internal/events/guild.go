package events

import (
	"regexp"

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
