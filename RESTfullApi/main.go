package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/clients"

	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/handlers"
)

const envFilePath = "../.env"

func main() {

	logger := log.New(os.Stdout, "mongoDBAtlas-api ", log.LstdFlags)

	client, err := clients.ConnectClient(logger, envFilePath)

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

	nameEndpoint := os.Getenv("NAME_ENDPOINT")

	getRouter.HandleFunc("/person/{id}", EndpointHandlerGet.GetPersonByIdEndpoint)
	getRouter.HandleFunc("/people", EndpointHandlerGet.GetPeopleEndpoint)
	getRouter.HandleFunc(fmt.Sprintf("/personName/{%v}", nameEndpoint), EndpointHandlerGet.GetPersonByNameEndpoint)

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/person", EndpointHandlerPost.CreatePersonEndpoint)
	postRouter.Use(EndpointHandlerPost.MiddlewareValidateProduct)

	bindAddress := os.Getenv("BIND_ADDRESS")

	if bindAddress == "" {
		bindAddress = "localhost:8080"
	}

	s := http.Server{
		Addr:         bindAddress,       // configure the bind address
		Handler:      router,            // set the default handler
		ErrorLog:     logger,            // set the logger for the server
		ReadTimeout:  5 * time.Second,   // max time to read request from the client
		WriteTimeout: 10 * time.Second,  // max time to write response to the client
		IdleTimeout:  120 * time.Second, // max time for connections using TCP Keep-Alive
	}

	logger.Fatal(s.ListenAndServe())

}
