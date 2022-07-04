package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"regexp"
	"time"

	"github.com/asianchinaboi/backendserver/logger"
)

type account struct {
	Id       int    `json:"userId"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type userInfoData struct {
	Icon     int    `json:"icon"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type session struct {
	Expires int64
	Id      int
}

type sessionToken struct {
	Expires int64  `json:"expires"`
	Id      int    `json:"userId"`
	Token   string `json:"token"`
}

type dataChange struct {
	Password string `json:"password"`
	Change   int    `json:"change"` // 0 for password, 1 for email, 2 for username
	NewData  string `json:"newData"`
}

const expireTime = 60 * 24 * time.Hour //token will expire in 2 months

func checkToken(token string) (*session, error) {
	var user session
	err := db.QueryRow("SELECT user_id, token_expires FROM tokens WHERE token=$1", token).Scan(&user.Id, &user.Expires)
	if err != nil && err == sql.ErrNoRows {
		return nil, errorInvalidToken
	} else if err != nil {
		return nil, err
	}
	if time.Now().Unix() > user.Expires {
		db.Exec("DELETE FROM tokens WHERE token=$1", token)
		return nil, errorExpiredToken
	}
	return &user, nil
}

func genToken(id int) (sessionToken, error) {
	var authData sessionToken
	err := db.QueryRow("SELECT token, token_expires FROM tokens WHERE user_id=$1", id).Scan(&authData.Token, &authData.Expires)
	if err != nil && err != sql.ErrNoRows {
		return authData, err
	} else if err == sql.ErrNoRows {
		authToken := generateRandString(16)
		authExpires := time.Now().Add(expireTime).UnixMilli()
		authData = sessionToken{Id: id, Expires: authExpires, Token: authToken}
		_, err = db.Exec("INSERT INTO tokens (user_id, token, token_expires) VALUES ($1, $2, $3)", id, authToken, authExpires)
		if err != nil {
			return authData, err
		}
	} else {
		authData.Id = id
	}
	return authData, nil

}

//make email validation
func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func userlogin(w http.ResponseWriter, r *http.Request) {
	log.WriteLog(logger.INFO, "Getting user")
	params := r.URL.Query()
	username := params.Get("username")
	password := params.Get("pwd")
	var acc account
	hashedpass := fmt.Sprintf("%x", sha256.Sum256([]byte(password+username)))
	if err := db.QueryRow("SELECT * FROM users WHERE username=$1 AND password=$2", username, hashedpass).Scan(&acc.Id, &acc.Email, &acc.Password, &acc.Username); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	//authToken := generateRandString(16)
	//log.WriteLog(logger.INFO, fmt.Sprintf("time to add %v", expireTime))
	//log.WriteLog(logger.INFO, fmt.Sprintf("time now unix %v", time.Now().Unix()))
	log.WriteLog(logger.INFO, "Generating token")

	authData, err := genToken(acc.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	result, err := json.Marshal(authData)

	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func createuser(w http.ResponseWriter, r *http.Request) {
	log.WriteLog(logger.INFO, "Creating User")
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
	//validation
	if acc.Email != "" && !validateEmail(acc.Email) {
		reportError(http.StatusBadRequest, w, errorInvalidEmail)
		return
	}
	usernameValid, err := regexp.MatchString("^[a-zA-Z0-9_]{6,32}$", acc.Username)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	passwordValid, err := regexp.MatchString(`^[\x00-\xFF]{6,64}$`, acc.Password)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}

	if !usernameValid || !passwordValid {
		reportError(http.StatusBadRequest, w, errorInvalidDetails)
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
	err = db.QueryRow("INSERT INTO users (email, password, username) VALUES ($1, $2, $3) RETURNING id", acc.Email, hashedpass, acc.Username).Scan(&acc.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	//create session for new user
	authData, err := genToken(acc.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	result, _ := json.Marshal(authData)
	log.WriteLog(logger.INFO, fmt.Sprintf("info about new user %v", authData))
	w.WriteHeader(http.StatusOK)
	w.Write(result)
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

	//validation
	var username string
	row := db.QueryRow("SELECT username FROM users WHERE id=$1", user.Id)
	if err := row.Err(); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	row.Scan(&username)
	checkPass := fmt.Sprintf("%x", sha256.Sum256([]byte(change.Password+username)))
	row = db.QueryRow("SELECT EXISTS (SELECT id FROM users WHERE password=$1)", checkPass)
	if err := row.Err(); err != nil && err != sql.ErrNoRows {
		reportError(http.StatusInternalServerError, w, err)
		return
	} else if err == sql.ErrNoRows {
		reportError(http.StatusBadRequest, w, errorInvalidDetails)
		return
	}

	switch change.Change {
	case 0:
		hashedpass := fmt.Sprintf("%x", sha256.Sum256([]byte(change.NewData+username)))
		_, err = db.Exec("UPDATE users SET password=$1 WHERE id=$2 AND password=$3", hashedpass, user.Id, change.Password)
		if err != nil {
			reportError(http.StatusBadRequest, w, err)
			return
		}
	case 1:
		_, err = db.Exec("UPDATE users SET email=$1 WHERE id=$2", change.NewData, user.Id)
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}

	case 2:
		_, err = db.Exec("UPDATE users SET username=$1 WHERE id=$2", change.NewData, user.Id)
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

func userInfo(w http.ResponseWriter, r *http.Request, user *session) {
	var acc userInfoData
	row := db.QueryRow("SELECT email, username FROM users WHERE id=$1", user.Id)
	if err := row.Err(); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	//placeholder for now
	acc.Icon = 0
	err := row.Scan(&acc.Email, &acc.Username)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	result, err := json.Marshal(acc)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}
