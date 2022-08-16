package main

import "errors"

var (
	errorToken              = errors.New("token is not provided")
	errorInvalidToken       = errors.New("token is invalid")
	errorExpiredToken       = errors.New("token has expired")
	errorNotInGuild         = errors.New("user is not in guild")
	errorAlreadyInGuild     = errors.New("user is already in guild or is banned")
	errorCantKickBanSelf    = errors.New("you can't kick or ban yourself")
	errorCantLeaveOwnGuild  = errors.New("you can't leave your own guild")
	errorUsernameExists     = errors.New("username already exists")
	errorEmailExists        = errors.New("email already exists")
	errorInvalidChange      = errors.New("invalid change option")
	errorGuildNotProvided   = errors.New("guild is not provided")
	errorNotGuildOwner      = errors.New("user is not owner")
	errorNoInvite           = errors.New("no invite provided")
	errorInvalidInvite      = errors.New("invalid invite provided")
	errorInviteLimitReached = errors.New("invite limit reached")
	errorGuildPoolNotExist  = errors.New("guild pool does not exist")
	errorInvalidEmail       = errors.New("invalid email")
	errorInvalidDetails     = errors.New("invalid details")
	errorNoMsgContent       = errors.New("no msg content")
	errorMsgTooLong         = errors.New("msg too long")
)
