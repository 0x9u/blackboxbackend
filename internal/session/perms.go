package session

import "github.com/asianchinaboi/backendserver/internal/errors"

func GetPerms(permId int, perms *Permissions) error {
	switch permId {
	case 1:
		perms.Admin = true
	case 2:
		perms.BanIP = true
	case 3:
		perms.Users.Get = true
	case 4:
		perms.Users.Edit = true
	case 5:
		perms.Users.Delete = true
	case 6:
		perms.Guilds.Get = true
	case 7:
		perms.Guilds.Edit = true
	case 8:
		perms.Guilds.Delete = true
	default:
		return errors.ErrInvalidPermission
	}
	return nil
}
