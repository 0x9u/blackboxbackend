package main

import (
	"database/sql"
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

const (
	host       = "localhost"
	port       = 5432
	user       = "postgres"
	password   = "1"
	dbname     = "chatapp"
	sslenabled = "disable"
)

/*
func reportError(w http.ResponseWriter, err error) {
	log.WriteLog(logger.ERROR, err.Error())
	w.Write([]byte("bad request"))
	w.WriteHeader(http.StatusBadRequest)
}
*/
var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func generateRandString(l int) string { //copied from stackoverflow
	b := make([]rune, l)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

func msgReset(w http.ResponseWriter, r *http.Request) {
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
	r.HandleFunc("/api/send", msgRecieve)
	r.HandleFunc("/api/msg", msgSend)
	r.HandleFunc("/api/ws", msgSocket)
	r.HandleFunc("/api/reset", msgReset)
	r.HandleFunc("/api/login", userlogin)
	r.HandleFunc("/api/signup", createuser)
	r.HandleFunc("/api/guild/create", createGuild)
	//make some function that grabs the images and videos based on "/files/*(put a random int here) format timestamp_(user_id)"
	//make some function that grabs user profiles based on "/user profiles/*(put a random int here (user id))"
	//make some function that grabs user profiles based on "/guild icons/*(put a random int here (guild id))"
	log.WriteLog(logger.INFO, "Handling requests")
	http.Handle("/", r)
	http.ListenAndServe(":8090", nil)
}

func initWs() {
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}
