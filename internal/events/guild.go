package events

import (
	"regexp"

	"github.com/asianchinaboi/backendserver/internal/errors"
)

type UserGuild struct {
	GuildId  int   `json:"guild:id"`
	UserId   int   `json:"user:id"` //Id to remove/add user
	UserData *User `json:"userData,omitempty"`
}

//used for guild settings, info, join info, update info
type Guild struct {
	OwnerId  int        `json:"owner:id,omitempty"`
	GuildId  int        `json:"id"`
	Name     string     `json:"name,omitempty"`
	Icon     int        `json:"icon,omitempty"`
	Unread   *UnreadMsg `json:"unread,omitempty"`
	SaveChat *bool      `json:"saveChat,omitempty"`
}

type Invite struct {
	Invite  string `json:"invite"`
	GuildId int    `json:"guild:id"`
}

func ValidateGuildInput(body *Guild) (errors.StatusCode, error) {

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
