package main

import (
	"crypto/sha256"
	"encoding/json"
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

func userlogin(w http.ResponseWriter, r *http.Request) {
	var acc account
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &acc); err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := db.QueryRow("SELECT * FROM users WHERE password=$1 AND email=$2", acc.Password, acc.Email).Scan(&acc); err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	authToken := generateRandString(16)
	authExpires := time.Now().Unix()
	tokens[authToken] = session{Id: acc.Id, Expires: authExpires}
	auth := identity{Token: authToken}
	result, err := json.Marshal(auth)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Write(result)
	w.WriteHeader(http.StatusAccepted)
}

func createuser(w http.ResponseWriter, r *http.Request) {
	var acc account
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	err = json.Unmarshal(bodyBytes, &acc)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	hashedpass := sha256.Sum256([]byte(acc.Password + acc.Username)) //just in case users have same password coincidentally
	_, err = db.Exec("INSERT INTO users (email, password, username) VALUES ($1, $2, $3)", acc.Email, string(hashedpass[:]), acc.Username)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
