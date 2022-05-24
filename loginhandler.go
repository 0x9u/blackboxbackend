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
