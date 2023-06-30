package errors

type StatusCode int

const (
	StatusInternalError StatusCode = iota
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
	StatusGuildPoolNotExist
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

	StatusAllFieldsEmpty
	StatusInvalidDetails
	StatusNotExist

	StatusCooldownActive

	StatusIpBanned

	StatusFileNotFound
	StatusFileInvalid
	StatusFileNoBytes
	StatusFileTooLarge

	StatusRouteParamInvalid

	StatusSessionTooManySessions

	StatusNotAuthorised
)
