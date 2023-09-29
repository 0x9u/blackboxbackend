package errors

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/logger"
)

type ErrCode int

const (
	StatusInternalError ErrCode = iota
	StatusBadRequest

	StatusAbsentToken
	StatusInvalidToken
	StatusExpiredToken

	StatusInvalidGuildName

	StatusNotInGuild
	StatusAlreadyInGuild
	StatusCantKickBanSelf
	StatusAlreadyBanned
	StatusUserNotBanned
	StatusCantLeaveOwnGuild
	StatusNotGuildAuthorised
	StatusAlreadyOwner
	StatusAlreadyAdmin

	StatusGuildSaveChatOn
	StatusGuildNotProvided
	StatusGuildPoolNotExist //not used
	StatusGuildNotExist
	StatusGuildIsDm

	StatusDmNotOpened
	StatusDmCannotDmSelf
	StatusDmNotExist

	StatusFriendBlocked
	StatusFriendAlreadyFriends
	StatusFriendAlreadyRequested
	StatusFriendRequestNotFound
	StatusFriendInvalid
	StatusFriendCannotRequest
	StatusFriendSelf

	StatusUserNotBlocked

	StatusUsernameExists
	StatusEmailExists
	StatusInvalidEmail
	StatusInvalidPass
	StatusInvalidUsername
	StatusUserNotFound

	StatusUserAlreadyAdmin
	StatusUserNotAdmin

	StatusNoInvite
	StatusInvalidInvite
	StatusInviteLimitReached

	StatusNoMsgContent
	StatusMsgTooLong
	StatusMsgNotExist
	StatusMsgUserBlocked

	StatusAllFieldsEmpty
	StatusInvalidDetails

	StatusCooldownActive

	StatusIpBanned

	StatusFileNotFound
	StatusFileInvalid
	StatusFileNoBytes
	StatusFileTooLarge

	StatusRouteParamInvalid

	StatusSessionTooManySessions //not used

	StatusNotAuthorised
)

func getHTTPStatusCode(errorCode ErrCode) int {
	switch errorCode {
	case StatusInternalError:
		return http.StatusInternalServerError
	case StatusBadRequest:
		return http.StatusBadRequest
	case StatusAbsentToken:
		return http.StatusForbidden
	case StatusInvalidToken:
		return http.StatusUnauthorized
	case StatusExpiredToken:
		return http.StatusUnauthorized
	case StatusInvalidGuildName:
		return http.StatusUnprocessableEntity
	case StatusNotInGuild:
		return http.StatusForbidden
	case StatusAlreadyInGuild:
		return http.StatusConflict
	case StatusCantKickBanSelf:
		return http.StatusForbidden
	case StatusAlreadyBanned:
		return http.StatusConflict
	case StatusUserNotBanned:
		return http.StatusConflict
	case StatusCantLeaveOwnGuild:
		return http.StatusForbidden
	case StatusNotGuildAuthorised:
		return http.StatusForbidden
	case StatusAlreadyOwner:
		return http.StatusConflict
	case StatusAlreadyAdmin:
		return http.StatusConflict
	case StatusGuildSaveChatOn:
		return http.StatusForbidden
	case StatusGuildNotProvided:
		return http.StatusBadRequest

	case StatusGuildPoolNotExist: // not used
		return http.StatusNotFound

	case StatusGuildNotExist:
		return http.StatusNotFound
	case StatusGuildIsDm:
		return http.StatusForbidden
	case StatusDmNotOpened:
		return http.StatusNotFound

	case StatusDmCannotDmSelf:
		return http.StatusForbidden
	case StatusDmNotExist:
		return http.StatusNotFound
	case StatusFriendBlocked:
		return http.StatusForbidden

	case StatusFriendAlreadyFriends:
		return http.StatusConflict

	case StatusFriendAlreadyRequested:
		return http.StatusConflict

	case StatusFriendRequestNotFound:
		return http.StatusNotFound

	case StatusFriendInvalid:
		return http.StatusBadRequest
	case StatusFriendCannotRequest:
		return http.StatusForbidden

	case StatusFriendSelf:
		return http.StatusForbidden
	case StatusUserNotBlocked:
		return http.StatusNotFound

	case StatusUsernameExists:
		return http.StatusBadRequest

	case StatusEmailExists:
		return http.StatusConflict
	case StatusInvalidEmail:
		return http.StatusBadRequest
	case StatusInvalidPass:
		return http.StatusBadRequest
	case StatusInvalidUsername:
		return http.StatusBadRequest
	case StatusUserNotFound:
		return http.StatusNotFound
	case StatusUserAlreadyAdmin:
		return http.StatusConflict
	case StatusUserNotAdmin:
		return http.StatusForbidden
	case StatusNoInvite:
		return http.StatusBadRequest
	case StatusInvalidInvite:
		return http.StatusBadRequest
	case StatusInviteLimitReached:
		return http.StatusForbidden
	case StatusNoMsgContent:
		return http.StatusBadRequest
	case StatusMsgTooLong:
		return http.StatusBadRequest
	case StatusMsgUserBlocked:
		return http.StatusForbidden
	case StatusAllFieldsEmpty:
		return http.StatusBadRequest
	case StatusInvalidDetails:
		return http.StatusUnprocessableEntity
	case StatusMsgNotExist:
		return http.StatusNotFound
	case StatusCooldownActive:
		return http.StatusTooManyRequests
	case StatusIpBanned:
		return http.StatusForbidden
	case StatusFileNotFound:
		return http.StatusNotFound
	case StatusFileInvalid:
		return http.StatusBadRequest
	case StatusFileNoBytes:
		return http.StatusBadRequest
	case StatusFileTooLarge:
		return http.StatusBadRequest
	case StatusRouteParamInvalid:
		return http.StatusBadRequest
	case StatusSessionTooManySessions: //not used
		return http.StatusForbidden
	case StatusNotAuthorised:
		return http.StatusForbidden
	default:
		logger.Warn.Printf("Unknown error code: %v\n", errorCode)
		return http.StatusInternalServerError
	}
}
