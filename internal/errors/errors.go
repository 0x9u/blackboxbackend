package errors

import (
	"errors"

	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/gin-gonic/gin"
)

type Body struct {
	Error  string  `json:"error"`
	Status ErrCode `json:"status"`
	Index  int     `json:"index,omitempty"`
}

var (

	//TOKEN

	ErrAbsentToken  = errors.New("token: not provided")
	ErrInvalidToken = errors.New("token: invalid")
	ErrExpiredToken = errors.New("token: expired")

	//USER GUILD

	ErrInvalidGuildName = errors.New("guild: invalid name")

	ErrNotInGuild         = errors.New("guild: user is not in guild")
	ErrAlreadyInGuild     = errors.New("guild: user is already in guild or is banned")
	ErrCantKickBanSelf    = errors.New("guild: you can't kick or ban yourself")
	ErrAlreadyBanned      = errors.New("guild: user is already banned")
	ErrUserNotBanned      = errors.New("guild: user is not banned")
	ErrCantLeaveOwnGuild  = errors.New("guild: you can't leave your own guild")
	ErrNotGuildAuthorised = errors.New("guild: user is not authorised")
	ErrAlreadyOwner       = errors.New("guild: already owner")
	ErrAlreadyAdmin       = errors.New("guild: already admin")

	//GUILD AND MISC

	ErrGuildNotProvided  = errors.New("guild: not provided")
	ErrGuildSaveChatOn   = errors.New("guild: save chat is on")
	ErrGuildPoolNotExist = errors.New("guild: pool does not exist")
	ErrGuildNotExist     = errors.New("guild: doesn't exist")
	ErrGuildIsDm         = errors.New("guild: is dm")

	//DM

	ErrDmNotOpened    = errors.New("dm: not opened")
	ErrDmCannotDmSelf = errors.New("dm: cannot dm self")
	ErrDmNotExist     = errors.New("dm: doesnt exists")

	//FRIND
	ErrFriendBlocked          = errors.New("friend: blocked")
	ErrFriendAlreadyFriends   = errors.New("friend: already friends")
	ErrFriendAlreadyRequested = errors.New("friend: already requested")
	ErrFriendRequestNotFound  = errors.New("friend: request not found")
	ErrFriendInvalid          = errors.New("friend: invalid friend")
	ErrFriendCannotRequest    = errors.New("friend: cannot request")
	ErrFriendSelf             = errors.New("friend: cannot add self")

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

	//GUILD ADMIN
	ErrUserAlreadyAdmin = errors.New("guild admin: user is already admin")
	ErrUserNotAdmin     = errors.New("guild admin: user is not admin")

	//INVITE

	ErrNoInvite           = errors.New("invite: none provided")
	ErrInvalidInvite      = errors.New("invite: invalid")
	ErrInviteLimitReached = errors.New("invite: limit reached")

	//MSG

	ErrNoMsgContent = errors.New("msg: no content")
	ErrMsgTooLong   = errors.New("msg: length too long")
	ErrMsgNotExist  = errors.New("msg: doesn't exist")

	//PATCH

	ErrAllFieldsEmpty = errors.New("patch: all fields are empty")
	ErrInvalidDetails = errors.New("patch: invalid details")

	//COOLDOWN

	ErrCooldownActive = errors.New("cooldown: cooldown is active")

	//IP
	ErrIpBanned = errors.New("ip: banned")

	//FILES
	ErrFileNotFound = errors.New("file: not found")
	ErrFileInvalid  = errors.New("file: invalid")
	ErrFileNoBytes  = errors.New("file: no bytes")
	ErrFileTooLarge = errors.New("file: too large")

	//ROUTES
	ErrRouteParamInvalid = errors.New("route: invalid param")

	//SESSION
	ErrNotAuthorised          = errors.New("session: not authorised")
	ErrInvalidPermission      = errors.New("session: invalid permission") //internal error
	ErrSessionDidntPass       = errors.New("session: didn't pass")        //internal error
	ErrSessionTooManySessions = errors.New("session: too many sessions")

	//CONTENT TYPE
	ErrNotSupportedContentType = errors.New("content type: not supported")
)

func SendErrorResponse(c *gin.Context, err error, errorCode ErrCode) {
	logger.Error.Println(err)
	c.JSON(getHTTPStatusCode(errorCode), Body{
		Error:  err.Error(),
		Status: errorCode,
	})
}
