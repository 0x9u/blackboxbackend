package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/asianchinaboi/backendserver/logger"
)

type account struct {
	Id       int    `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type identity struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

type session struct {
	Expires int64
	Id      int
}

type dataChange struct {
	Password string `json:"Password"`
	Change   int    `json:"Change"` // 0 for password, 1 for email, 2 for username
	New      string `json:"New"`
}

const expireTime = 60 * 24 * time.Hour //token will expire in 2 months

func checkToken(token string) (*session, error) {
	user, ok := tokens[token]
	if !ok {
		return nil, errorInvalidToken
	}
	if time.Now().Unix() > user.Expires {
		return nil, errorExpiredToken
	}
	return &user, nil
}

func userlogin(w http.ResponseWriter, r *http.Request) {
	var acc account
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if err := json.Unmarshal(bodyBytes, &acc); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	hashedpass := fmt.Sprintf("%x", sha256.Sum256([]byte(acc.Password+acc.Username)))
	if err := db.QueryRow("SELECT * FROM users WHERE username=$1 AND password=$2 AND email=$3", acc.Username, hashedpass, acc.Email).Scan(&acc.Id, &acc.Email, &acc.Password, &acc.Username); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	authToken := generateRandString(16)
	log.WriteLog(logger.INFO, fmt.Sprintf("time to add %v", expireTime))
	log.WriteLog(logger.INFO, fmt.Sprintf("time now unix %v", time.Now().Unix()))
	authExpires := time.Now().Add(expireTime).Unix()
	tokens[authToken] = session{Id: acc.Id, Expires: authExpires}
	auth := identity{Token: authToken, Expires: authExpires}
	result, err := json.Marshal(auth)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func createuser(w http.ResponseWriter, r *http.Request) {
	var acc account
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	log.WriteLog(logger.INFO, fmt.Sprintf("body %v", string(bodyBytes)))
	err = json.Unmarshal(bodyBytes, &acc)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var username string
	err = db.QueryRow("SELECT username FROM users WHERE username=$1", acc.Username).Scan(&username)
	if err != nil && err != sql.ErrNoRows {
		reportError(http.StatusInternalServerError, w, err)
		return
	} else if err != sql.ErrNoRows {
		reportError(http.StatusBadRequest, w, errorUsernameExists) //username already exists
		return
	}
	hashedpass := fmt.Sprintf("%x", sha256.Sum256([]byte(acc.Password+acc.Username))) //just in case users have same password coincidentally
	log.WriteLog(logger.INFO, hashedpass)
	_, err = db.Exec("INSERT INTO users (email, password, username) VALUES ($1, $2, $3)", acc.Email, hashedpass, acc.Username)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func changeDetails(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var change dataChange
	err = json.Unmarshal(bodyBytes, &change)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	switch change.Change {
	case 0:
		var username string
		row := db.QueryRow("SELECT username FROM users WHERE id=$1", user.Id)
		if err := row.Err(); err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		row.Scan(&username)
		hashedpass := fmt.Sprintf("%x", sha256.Sum256([]byte(change.Password+username)))
		_, err = db.Exec("UPDATE users SET password=$1 WHERE id=$2 AND password=$3", hashedpass, user.Id, change.Password)
		if err != nil {
			reportError(http.StatusBadRequest, w, err)
			return
		}
	case 1:
		_, err = db.Exec("UPDATE users SET email=$1 WHERE id=$2", change.New, user.Id)
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}

	case 2:
		_, err = db.Exec("UPDATE users SET username=$1 WHERE id=$2", change.New, user.Id)
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
	default:
		reportError(http.StatusBadRequest, w, errorInvalidChange)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
