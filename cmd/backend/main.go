package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/asianchinaboi/backendserver/internal/api"
	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/schedule"

	_ "net/http/pprof"
)

func main() {

	logger.Info.Println("Handling requests")

	if err := os.Mkdir("uploads", os.ModePerm); err != nil {
		logger.Fatal.Println(err)
	}
	if err := os.Mkdir("uploads/user", os.ModePerm); err != nil {
		logger.Fatal.Println(err)
	}
	if err := os.Mkdir("uploads/msg", os.ModePerm); err != nil {
		logger.Fatal.Println(err)
	}
	if err := os.Mkdir("uploads/guild", os.ModePerm); err != nil {
		logger.Fatal.Println(err)
	}
	defer db.Db.Close()

	server := api.StartServer()
	schedule.Start()

	go func() {
		if err := server.ListenAndServe(); err != nil {
			logger.Fatal.Println(err)
		}
	}()

	c := make(chan os.Signal, 1) //listen for cancellation
	signal.Notify(c, os.Interrupt)
	<-c //pause code here until interrupted

	logger.Info.Println("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), config.Config.Server.Timeout.Server)
	defer cancel()

	server.Shutdown(ctx)
}
