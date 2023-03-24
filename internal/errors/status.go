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
	StatusNotGuildOwner

	StatusGuildSaveChatOn
	StatusGuildNotProvided
	StatusGuildPoolNotExist
	StatusGuildNotExist

	StatusDMNotOpened
	StatusDmAlreadyExists

	StatusFriendBlocked
	StatusFriendAlreadyFriends
	StatusFriendAlreadyRequested
	StatusFriendRequestNotFound
	StatusFriendInvalid

	StatusUserNotBlocked

	StatusUsernameExists
	StatusEmailExists
	StatusInvalidEmail
	StatusInvalidPass
	StatusInvalidUsername
	StatusUserNotFound

	StatusNoInvite
	StatusInvalidInvite
	StatusInviteLimitReached

	StatusNoMsgContent
	StatusMsgTooLong

	StatusAllFieldsEmpty
	StatusInvalidDetails
	StatusNotExists

	StatusCooldownActive

	StatusIpBanned

	StatusFileNotFound

	StatusRouteParamInvalid

	StatusNotAuthorised
)
