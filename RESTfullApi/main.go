package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"RESTfullAPi-Golang/RESTfullApi/clients"

	"RESTfullAPi-Golang/RESTfullApi/handlers"
)

func main() {

	logger := log.New(os.Stdout, "mongoDBAtlas-api ", log.LstdFlags)

	client := clients.ConnectClient(logger)

	collection := client.Database("thepolyglotdeveloper").Collection("people")

	GetPersonByIdEndpoint := handlers.NewGetPersonByIdEndpoint(logger, collection)
	CreatePersonEndpoint := handlers.NewCreatePersonEndpoint(logger, collection)
	GetPeopleEndpoint := handlers.NewGetPeopleEndpoint(logger, collection)
	GetPersonByNameEndpoint := handlers.NewGetPersonByNameEndpoint(logger, collection)

	logger.Println("Starting the application...")

	clients.CtrlCHandler(client, logger)

	defer func() {
		clients.DisconnectClient(client, logger)
	}()

	router := mux.NewRouter()

	getRouter := router.Methods(http.MethodGet).Subrouter()

	getRouter.HandleFunc("/person/{id}", GetPersonByIdEndpoint.ServeHTTP)
	getRouter.HandleFunc("/people", GetPeopleEndpoint.ServeHTTP)
	getRouter.HandleFunc("/personName/{name}", GetPersonByNameEndpoint.ServeHTTP)

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/person", CreatePersonEndpoint.ServeHTTP)

	http.ListenAndServe("localhost:8080", router)

}
