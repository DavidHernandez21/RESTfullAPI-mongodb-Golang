package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/clients"

	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/handlers"
)

func main() {

	logger := log.New(os.Stdout, "mongoDBAtlas-api ", log.LstdFlags)

	client, err := clients.ConnectClient(logger)

	if err != nil {
		logger.Fatalf("Error while connecting to the mongoDB client: %v", err)
	}

	collection := client.Database("thepolyglotdeveloper").Collection("people")

	EndpointHandler := handlers.NewEndpointHandler(logger, collection)

	logger.Println("Starting the application...")

	clients.CtrlCHandler(client, logger)

	defer func() {
		clients.DisconnectClient(client, logger)
	}()

	router := mux.NewRouter()

	getRouter := router.Methods(http.MethodGet).Subrouter()

	getRouter.HandleFunc("/person/{id}", EndpointHandler.GetPersonByIdEndpoint)
	getRouter.HandleFunc("/people", EndpointHandler.GetPeopleEndpoint)
	getRouter.HandleFunc("/personName/{name}", EndpointHandler.GetPersonByNameEndpoint)

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/person", EndpointHandler.CreatePersonEndpoint)
	postRouter.Use(EndpointHandler.MiddlewareValidateProduct)

	http.ListenAndServe("localhost:8080", router)

}
