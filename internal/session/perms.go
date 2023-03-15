package session

import "github.com/asianchinaboi/backendserver/internal/errors"

func getPerms(permId int, user *Session) error {
	switch permId {
	case 1:
		user.Perms.Admin = true
	case 2:
		user.Perms.BanIP = true
	case 3:
		user.Perms.Users.Get = true
	case 4:
		user.Perms.Users.Edit = true
	case 5:
		user.Perms.Users.Delete = true
	case 6:
		user.Perms.Guilds.Get = true
	case 7:
		user.Perms.Guilds.Edit = true
	case 8:
		user.Perms.Guilds.Delete = true
	default:
		return errors.ErrInvalidPermission
	}
	return nil
}
