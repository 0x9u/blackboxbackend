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

	defer func() {
		if r := recover(); r != nil {
			logger.Fatal.Printf("Panic: %v\n", r)
			os.Exit(1)
		}
	}()

	if err := os.MkdirAll("uploads/user", os.ModePerm); err != nil {
		logger.Fatal.Panicln(err)
	}
	if err := os.MkdirAll("uploads/msg", os.ModePerm); err != nil {
		logger.Fatal.Panicln(err)
	}
	if err := os.MkdirAll("uploads/guild", os.ModePerm); err != nil {
		logger.Fatal.Panicln(err)
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
