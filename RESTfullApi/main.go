package main

import (
	"log"
	"net/http"
	"os"
	"time"

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

	EndpointHandlerPost := handlers.NewEndpointHandler(logger, collection)

	EndpointHandlerGet := handlers.NewEndpointHandler(logger, collection, handlers.WithTimeout(10*time.Second))

	logger.Println("Starting the application...")

	clients.CtrlCHandler(client, logger)

	defer func() {
		err := clients.DisconnectClient(client, logger)
		if err != nil {
			logger.Fatalf("Error disconnecting the client: %v\n", err)
		}
	}()

	router := mux.NewRouter()

	getRouter := router.Methods(http.MethodGet).Subrouter()

	getRouter.HandleFunc("/person/{id}", EndpointHandlerGet.GetPersonByIdEndpoint)
	getRouter.HandleFunc("/people", EndpointHandlerGet.GetPeopleEndpoint)
	getRouter.HandleFunc("/personName/{name}", EndpointHandlerGet.GetPersonByNameEndpoint)

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/person", EndpointHandlerPost.CreatePersonEndpoint)
	postRouter.Use(EndpointHandlerPost.MiddlewareValidateProduct)

	http.ListenAndServe("localhost:8080", router)

}
