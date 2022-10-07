package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	_ "net/http/pprof"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"

	"github.com/asianchinaboi/backendserver/logger"
)

var (
	log         *logger.Logger
	db          *sql.DB
	upgrader    websocket.Upgrader
	clients     = make(map[string]brcastEvents)         // string (unique id) : channel
	clientAlias = make(map[int]map[string]brcastEvents) // alias (string) : id : channel
	pools       = make(map[int]*pool)                   //guild id : guild channel
	characters  = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
) // convert or look more into the sync map in golang

const ( //settings
	host       = "localhost"
	port       = 5432
	user       = "postgres"
	password   = "1"
	dbname     = "chatapp"
	sslenabled = "disable"
)

type statusInfo struct {
	ClientNumber int `json:"clientNumber"`
	GuildNumber  int `json:"guildNumber"`
	MsgNumber    int `json:"msgNumber"`
}

type errorInfo struct {
	Error string `json:"error"`
}

func reportError(status int, w http.ResponseWriter, err error) {
	log.WriteLog(logger.ERROR, err.Error())
	w.WriteHeader(status)
	errorData := errorInfo{
		Error: err.Error(),
	}
	bodyBytes, err := json.Marshal(errorData)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	w.Write(bodyBytes)
}

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

func staticFiles(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path = filepath.Join("build", path)
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		http.ServeFile(w, r, filepath.Join("build", "index.html"))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.FileServer(http.Dir("build")).ServeHTTP(w, r)
}

func showStatus(w http.ResponseWriter, r *http.Request) { //debugging
	row := db.QueryRow("SELECT COUNT(*) FROM messages")
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, row.Err())
		return
	}
	var msgNumber int
	row.Scan(&msgNumber)
	row = db.QueryRow("SELECT COUNT(*) FROM guilds")
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, row.Err())
		return
	}
	var guildNumber int
	row.Scan(&guildNumber)

	status := statusInfo{
		ClientNumber: len(clients),
		GuildNumber:  guildNumber,
		MsgNumber:    msgNumber,
	}
	bodyBytes, err := json.Marshal(status)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
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

	defer func() { //logs fatal panics
		if r := recover(); r != nil {
			log.WriteLog(logger.FATAL, fmt.Sprintf("%v", r))
			os.Exit(1)
		}
	}()

	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	r.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)

	api.HandleFunc("/msg", middleWare(msgRecieve)).Methods("POST")
	api.HandleFunc("/msg", middleWare(msgEdit)).Methods("PUT")
	api.HandleFunc("/msg", middleWare(msgHistory)).Methods("GET")
	api.HandleFunc("/msg", middleWare(msgDelete)).Methods("DELETE")

	api.HandleFunc("/guild", middleWare(createGuild)).Methods("POST")
	api.HandleFunc("/guild", middleWare(deleteGuild)).Methods("DELETE")
	api.HandleFunc("/guild", middleWare(getGuild)).Methods("GET") //might send guilds to client through websocket
	api.HandleFunc("/guild", middleWare(editGuild)).Methods("PUT")
	api.HandleFunc("/guild/setting", middleWare(getGuildSettings)).Methods("GET")
	api.HandleFunc("/guild/users", middleWare(getGuildUsers)).Methods("GET")
	api.HandleFunc("/guild/join", middleWare(joinGuild)).Methods("POST")
	api.HandleFunc("/guild/ban", middleWare(banGuildUser)).Methods("POST")
	api.HandleFunc("/guild/ban", middleWare(unbanUser)).Methods("PUT")     //unban
	api.HandleFunc("/guild/ban", middleWare(getBannedList)).Methods("GET") //get all banned users
	api.HandleFunc("/guild/kick", middleWare(kickGuildUser)).Methods("POST")
	api.HandleFunc("/guild/leave", middleWare(leaveGuild)).Methods("POST") //leave guild
	api.HandleFunc("/guild/deletemsg", middleWare(clearGuildMsg)).Methods("DELETE")

	api.HandleFunc("/invite", middleWare(genGuildInvite)).Methods("POST")
	api.HandleFunc("/invite", middleWare(getGuildInvite)).Methods("GET")
	api.HandleFunc("/invite", middleWare(deleteInvGuild)).Methods("DELETE")

	api.HandleFunc("/ws", webSocket)   //make middleware later for token validation
	api.HandleFunc("/reset", msgReset) //dangerous

	api.HandleFunc("/user", userlogin).Methods("GET")
	api.HandleFunc("/user", createuser).Methods("POST")
	api.HandleFunc("/user", middleWare(changeDetails)).Methods("PUT")
	api.HandleFunc("/user/info", middleWare(userInfo)).Methods("GET")
	api.HandleFunc("/user/deletemsg", middleWare(clearUserMsg)).Methods("DELETE")

	api.HandleFunc("/status", showStatus).Methods("GET")
	//make some function that grabs the images and videos based on "/files/*(put a random int here) format timestamp_(user_id)"
	//make some function that grabs user profiles based on "/user profiles/*(put a random int here (user id))"
	//make some function that grabs user profiles based on "/guild icons/*(put a random int here (guild id))"
	//Allow any connection /*/ for client side routing

	// \A\/[^api](.*) is a regex that matches everything except api
	//r.HandleFunc(`\A\/[^api](.*)`, )
	//buildHandler := http.FileServer(http.Dir("build"))
	//r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./build/static/"))))

	//r.PathPrefix("/").Handler(http.FileServer(http.Dir("./build"))).Subrouter().NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	http.ServeFile(w, r, "./build/index.html")
	//})

	r.PathPrefix("/").HandlerFunc(staticFiles)

	srv := &http.Server{ //server settings
		Addr: "0.0.0.0:8090",
		//prevents ddos attacks
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 15,
		Handler: handlers.CORS(
			handlers.AllowedHeaders([]string{"content-type", "Auth-Token", ""}), //took some time to figure out middleware problem
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS", "DELETE"}),
			handlers.AllowCredentials(),
		)(r),
	}

	log.WriteLog(logger.INFO, "Handling requests")

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.WriteLog(logger.FATAL, err.Error())
		}
	}()

	c := make(chan os.Signal, 1) //listen for cancellation
	signal.Notify(c, os.Interrupt)
	<-c //pause code here until interrupted

	log.WriteLog(logger.INFO, "Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	srv.Shutdown(ctx)
}

func initWs() {
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}
