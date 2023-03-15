package errors

import "errors"

type Body struct {
	Error  string     `json:"error"`
	Status StatusCode `json:"status"`
	Index  int        `json:"index,omitempty"`
}

var (

	//TOKEN

	ErrAbsentToken  = errors.New("token: not provided")
	ErrInvalidToken = errors.New("token: invalid")
	ErrExpiredToken = errors.New("token: expired")

	//USER GUILD

	ErrInvalidGuildName = errors.New("guild: invalid name")

	ErrNotInGuild        = errors.New("guild: user is not in guild")
	ErrAlreadyInGuild    = errors.New("guild: user is already in guild or is banned")
	ErrCantKickBanSelf   = errors.New("guild: you can't kick or ban yourself")
	ErrAlreadyBanned     = errors.New("guild: user is already banned")
	ErrUserNotBanned     = errors.New("guild: user is not banned")
	ErrCantLeaveOwnGuild = errors.New("guild: you can't leave your own guild")
	ErrNotGuildOwner     = errors.New("guild: user is not owner")

	//GUILD AND MISC

	ErrGuildNotProvided  = errors.New("guild: not provided")
	ErrGuildSaveChatOn   = errors.New("guild: save chat is on")
	ErrGuildPoolNotExist = errors.New("guild: pool does not exist")
	ErrGuildNotExist     = errors.New("guild: doesn't exist")

	//DM

	ErrDMNotOpened     = errors.New("dm: not opened")
	ErrDmAlreadyExists = errors.New("dm: already exists")

	//FRIND
	ErrFriendBlocked          = errors.New("friend: blocked")
	ErrFriendAlreadyFriends   = errors.New("friend: already friends")
	ErrFriendAlreadyRequested = errors.New("friend: already requested")
	ErrFriendRequestNotFound  = errors.New("friend: request not found")
	ErrFriendInvalid          = errors.New("friend: invalid friend")

	//BLOCKED
	ErrUserNotBlocked = errors.New("blocked: user not blocked")

	//USER

	ErrUsernameExists     = errors.New("user: username already exists")
	ErrEmailExists        = errors.New("user: email already exists")
	ErrInvalidEmail       = errors.New("user: invalid email")
	ErrInvalidPass        = errors.New("user: invalid password")
	ErrInvalidUsername    = errors.New("user: invalid username")
	ErrUserNotFound       = errors.New("user: user not found")
	ErrUserClientNotExist = errors.New("user: client does not exist")

	//INVITE

	ErrNoInvite           = errors.New("invite: none provided")
	ErrInvalidInvite      = errors.New("invite: invalid")
	ErrInviteLimitReached = errors.New("invite: limit reached")

	//MSG

	ErrNoMsgContent = errors.New("msg: no content")
	ErrMsgTooLong   = errors.New("msg: length too long")

	//PATCH

	ErrAllFieldsEmpty = errors.New("patch: all fields are empty")
	ErrInvalidDetails = errors.New("patch: invalid details")
	ErrNotExists      = errors.New("patch: doesn't exist")

	//COOLDOWN

	ErrCooldownActive = errors.New("cooldown: cooldown is active")

	//IP
	ErrIpBanned = errors.New("ip: banned")

	//ROUTES
	ErrRouteParamInvalid = errors.New("route: invalid param")

	//SESSION
	ErrNotAuthorised     = errors.New("session: not authorised")
	ErrInvalidPermission = errors.New("session: invalid permission") //internal error
	ErrSessionDidntPass  = errors.New("session: didn't pass")        //internal error
)
