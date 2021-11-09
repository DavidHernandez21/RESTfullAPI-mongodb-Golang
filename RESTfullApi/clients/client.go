package clients

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"RESTfullAPi-Golang/RESTfullApi/timer"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectClient(logger *log.Logger) *mongo.Client {

	stop := timer.StartTimer("ConnectClient", logger)

	defer stop()

	if err := godotenv.Load("../.env"); err != nil {
		logger.Println("No .env file found")
	}

	uri := os.Getenv("MONGODB_URI_WO_DATABASE")

	if uri == "" {
		logger.Fatal("You must set your 'MONGODB_URI' environmental variable. See\n\t https://docs.mongodb.com/drivers/go/current/usage-examples/")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		logger.Fatalf("Error while connecting to the mongoDB client: %v", err)
	}

	return client
}

func DisconnectClient(client *mongo.Client, logger *log.Logger) {

	stop := timer.StartTimer("DisconnectClient", logger)

	defer stop()

	if err := client.Disconnect(context.TODO()); err != nil {
		panic(err)
	}

	logger.Println("mongo client disconnected")

}

func CtrlCHandler(client *mongo.Client, logger *log.Logger) {

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		logger.Printf("- Ctrl+C pressed, exiting\n Signal recieved: %v\n", sig)
		DisconnectClient(client, logger)
		os.Exit(0)
	}()
}
