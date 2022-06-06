package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"

	"github.com/asianchinaboi/backendserver/logger"
)

var (
	log      *logger.Logger
	db       *sql.DB
	upgrader websocket.Upgrader
	clients  = make(map[int]brcastEvents) // id : channel
	tokens   = make(map[string]session)   // tokens : user ids
)

var (
	errorToken            = errors.New("token is not provided")
	errorInvalidToken     = errors.New("token is invalid")
	errorExpiredToken     = errors.New("token has expired")
	errorNotInGuild       = errors.New("user is not in guild")
	errorUsernameExists   = errors.New("username already exists")
	errorInvalidChange    = errors.New("invalid change option")
	errorGuildNotProvided = errors.New("guild is not provided")
)

const (
	host       = "localhost"
	port       = 5432
	user       = "postgres"
	password   = "1"
	dbname     = "chatapp"
	sslenabled = "disable"
)

func reportError(status int, w http.ResponseWriter, err error) {
	log.WriteLog(logger.ERROR, err.Error())
	w.WriteHeader(status)
	w.Write([]byte("bad request"))
}

var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func generateRandString(l int) string { //copied from stackoverflow
	b := make([]rune, l)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

func msgReset(w http.ResponseWriter, r *http.Request) { //dangerous remove when finished
	q := `
		DELETE FROM messages;
		ALTER SEQUENCE messages_id_seq RESTART WITH 1;
	`
	_, err := db.Exec(q)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("bad request"))
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var file *os.File
	log, file = logger.NewLogger()
	defer file.Close()

	var err error
	loginInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", host, port, user, password, dbname, sslenabled)
	db, err = sql.Open("postgres", loginInfo)

	initWs()

	if err != nil {
		log.WriteLog(logger.FATAL, err.Error())
	}
	defer db.Close()

	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/msg", middleWare(msgRecieve)).Methods("POST")
	api.HandleFunc("/msg", middleWare(msgHistory)).Methods("GET")
	api.HandleFunc("/msg", middleWare(msgDelete)).Methods("DELETE")
	api.HandleFunc("/guild", middleWare(createGuild)).Methods("POST")
	api.HandleFunc("/guild", middleWare(deleteGuild)).Methods("DELETE")
	api.HandleFunc("/invite", middleWare(genGuildInvite)).Methods("POST")
	api.HandleFunc("/invite", middleWare(deleteInvGuild)).Methods("DELETE")
	api.HandleFunc("/ws", webSocket)   //make middleware later for token validation
	api.HandleFunc("/reset", msgReset) //dangerous
	api.HandleFunc("/user", userlogin).Methods("GET")
	api.HandleFunc("/user", createuser).Methods("POST")
	api.HandleFunc("/user", middleWare(changeDetails)).Methods("PUT")
	//make some function that grabs the images and videos based on "/files/*(put a random int here) format timestamp_(user_id)"
	//make some function that grabs user profiles based on "/user profiles/*(put a random int here (user id))"
	//make some function that grabs user profiles based on "/guild icons/*(put a random int here (guild id))"
	//Allow any connection /*/ for client side routing
	log.WriteLog(logger.INFO, "Handling requests")
	http.Handle("/", r)
	log.WriteLog(logger.FATAL, http.ListenAndServe(":8090", nil).Error())
}

func initWs() {
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}
