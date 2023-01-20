package session

import (
	"database/sql"
	"encoding/hex"
	"math/rand"
	"time"

	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
)

const (
	tokenLength = 32
)

var (
	characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
)

func CheckToken(token string) (*Session, error) {
	var user Session
	err := db.Db.QueryRow("SELECT user_id, token_expires FROM tokens WHERE token=$1", token).Scan(&user.Id, &user.Expires)
	if err != nil && err == sql.ErrNoRows {
		return nil, errors.ErrInvalidToken
	} else if err != nil {
		return nil, err
	}
	if time.Now().Unix() > user.Expires {
		db.Db.Exec("DELETE FROM tokens WHERE token=$1", token)
		return nil, errors.ErrExpiredToken
	}

	//get permissions

	//each perm overrides each other
	//e.g if admin is enabled and the user had multiple roles
	//then admin is enabled for the user even if other roles dont mention admin

	rows, err := db.Db.Query(`SELECT DISTINCT r.permission_id FROM userroles u INNER JOIN rolepermissions r ON u.role_id = r.role_id WHERE user_id = $1 `, user.Id)
	if err != nil {
		return nil, err
	}
	user.Perms = &Permissions{}
	defer rows.Close()
	for rows.Next() {
		var roleId int
		if err := rows.Scan(&roleId); err != nil {
			return nil, err
		}
		if err := getPerms(roleId, &user); err != nil {
			return nil, err
		}

	}
	return &user, nil
}

func GenToken(id int) (Session, error) {
	var authData Session
	//delete token if expired
	_, err := db.Db.Exec("DELETE FROM tokens WHERE user_id=$1 AND token_expires < $2", id, time.Now().UnixMilli())
	if err != nil {
		return Session{}, err
	}
	err = db.Db.QueryRow("SELECT token, token_expires FROM tokens WHERE user_id=$1", id).Scan(&authData.Token, &authData.Expires)
	if err != nil && err != sql.ErrNoRows {
		return authData, err
	} else if err == sql.ErrNoRows {
		authToken, err := generateSecureToken(tokenLength)
		if err != nil {
			return Session{}, err
		}
		authExpires := time.Now().Add(config.Config.User.TokenExpireTime).UnixMilli()
		authData = Session{Id: id, Expires: authExpires, Token: authToken}
		_, err = db.Db.Exec("INSERT INTO tokens (user_id, token, token_expires) VALUES ($1, $2, $3)", id, authToken, authExpires)
		if err != nil {
			return Session{}, err
		}
	} else {
		authData.Id = id
	}
	return authData, nil

}

func generateSecureToken(l int) (string, error) { //also copied from stackoverflow probs still insecure
	token := make([]byte, l)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}

func GenerateRandString(l int) string { //copied from stackoverflow (insecure and shit dont use for token)
	b := make([]rune, l)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
