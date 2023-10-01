package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/clients"
	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/observability"

	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/handlers"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prometheus/client_golang/prometheus"

	"net/http/pprof"
	// _ "net/http/pprof"
)

var (
	envFilePath string
	timeout     time.Duration
)

func init() {
	flag.StringVar(&envFilePath, "envFilePath", "../.env", "path to .env file")
	flag.DurationVar(&timeout, "timeout", 10, "timeout in seconds")

}

func main() {

	flag.Parse()

	logger := log.New(os.Stdout, "mongoDBAtlas-api ", log.LstdFlags)

	if err := prometheus.Register(observability.TotalRequests); err != nil {
		logger.Println("Faled to register totalRequests:", err)

	}
	if err := prometheus.Register(observability.ResponseStatus); err != nil {
		logger.Println("Faled to register responseStatus:", err)

	}
	if err := prometheus.Register(observability.HTTPDuration); err != nil {
		logger.Println("Faled to register httpDuration:", err)

	}

	client, err := clients.ConnectClient(logger, envFilePath)

	if err != nil {
		logger.Fatalf("Error while connecting to the mongoDB client: %v", err)
	}

	collection := client.Database("thepolyglotdeveloper").Collection("people")

	EndpointHandlerPost := handlers.NewEndpointHandler(logger, collection)

	EndpointHandlerGet := handlers.NewEndpointHandler(logger, collection, handlers.WithTimeout(10*time.Second))

	router := mux.NewRouter()

	// debug pprof
	router.HandleFunc("/debug/pprof/", http.HandlerFunc(pprof.Index))
	router.HandleFunc("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	router.HandleFunc("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	router.HandleFunc("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))
	// allocs
	router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	router.Use(observability.PrometheusMiddleware)

	getRouter := router.Methods(http.MethodGet).Subrouter()

	nameEndpoint := os.Getenv("NAME_ENDPOINT")

	getRouter.HandleFunc("/person/{id}", EndpointHandlerGet.GetPersonByIdEndpoint)
	getRouter.HandleFunc("/people", EndpointHandlerGet.GetPeopleEndpoint)
	getRouter.HandleFunc(fmt.Sprintf("/personName/{%v}", nameEndpoint), EndpointHandlerGet.GetPersonByNameEndpoint)
	getRouter.Handle(os.Getenv("METRICS_ENDPOINT"), promhttp.Handler())

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/person", EndpointHandlerPost.CreatePersonEndpoint)
	postRouter.Use(EndpointHandlerPost.MiddlewareValidateProduct)

	delRouter := router.Methods(http.MethodDelete).Subrouter()
	delRouter.HandleFunc("/person/{id}", EndpointHandlerPost.DeletePersonByIdEndpoint)

	updateRouter := router.Methods(http.MethodPut).Subrouter()
	updateRouter.HandleFunc("/person/{id}", EndpointHandlerPost.UpdatePersonByIdEndpoint)
	updateRouter.Use(EndpointHandlerPost.MiddlewareValidateUpdateRequest)

	const BIND_ADDRESS = "BIND_ADDRESS"
	bindAddress := os.Getenv(BIND_ADDRESS)

	if bindAddress == "" {
		const Localhost = "127.0.0.1:8080"
		bindAddress = Localhost
	}

	s := http.Server{
		Addr:         bindAddress,       // configure the bind address
		Handler:      router,            // set the default handler
		ErrorLog:     logger,            // set the logger for the server
		ReadTimeout:  30 * time.Second,  // max time to read request from the client
		WriteTimeout: 300 * time.Second, // max time to write response to the client
		IdleTimeout:  120 * time.Second, // max time for connections using TCP Keep-Alive
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	clients.CtrlCHandler(ctx, client, logger, &s)

	const SERVER_STARTING = "Starting server on port %s"
	port := strings.Split(bindAddress, ":")[1]
	logger.Printf(SERVER_STARTING, port)

	err = s.ListenAndServe()
	if err == http.ErrServerClosed {
		logger.Println("Server closed under request")
	}

}
