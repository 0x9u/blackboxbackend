package main

import "errors"

var (
	errorToken             = errors.New("token is not provided")
	errorInvalidToken      = errors.New("token is invalid")
	errorExpiredToken      = errors.New("token has expired")
	errorNotInGuild        = errors.New("user is not in guild")
	errorUsernameExists    = errors.New("username already exists")
	errorInvalidChange     = errors.New("invalid change option")
	errorGuildNotProvided  = errors.New("guild is not provided")
	errorNotGuildOwner     = errors.New("user is not owner")
	errorNoInvite          = errors.New("no invite provided")
	errorInvalidInvite     = errors.New("invalid invite provided")
	errorGuildPoolNotExist = errors.New("guild pool does not exist")
	errorInvalidEmail      = errors.New("invalid email")
	errorInvalidDetails    = errors.New("invalid details")
)
