package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gapi-org/gapi"
)

type HelloIn struct {
	Name string `query:"name" default:"world" doc:"Name to greet"`
}

type HelloOut struct {
	Message string `json:"message"`
}

func Hello(ctx context.Context, in HelloIn) (HelloOut, error) {
	return HelloOut{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
	app := gapi.New(gapi.Config{
		Title:   "Hello API",
		Version: "0.1.0",
	})

	gapi.Get[HelloIn, HelloOut](
		app,
		"/hello",
		Hello,
		gapi.OperationID("hello"),
		gapi.Summary("Say hello"),
		gapi.Tags("hello"),
	)

	log.Fatal(http.ListenAndServe(":8080", app))
}
