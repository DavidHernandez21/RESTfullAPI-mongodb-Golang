package clients

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/timer"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func loadEnvFile(logger *log.Logger, filePath string) error {

	if err := godotenv.Load(filePath); err != nil {
		logger.Println("No .env file found")
		return err
	}

	return nil

}

func ConnectClient(logger *log.Logger, envFilePath string) (*mongo.Client, error) {

	stop := timer.StartTimer("ConnectClient", logger)

	defer stop()

	if err := loadEnvFile(logger, envFilePath); err != nil {
		return nil, err
	}

	uri := os.Getenv("MONGODB_URI_WO_DATABASE")

	if uri == "" {
		return nil, errors.New("you must set your 'MONGODB_URI' environmental variable. See\n\t https://docs.mongodb.com/drivers/go/current/usage-examples/")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func DisconnectClient(client *mongo.Client, logger *log.Logger) error {

	stop := timer.StartTimer("DisconnectClient", logger)

	defer stop()

	if err := client.Disconnect(context.TODO()); err != nil {
		return err
	}

	logger.Println("mongo client disconnected")

	return nil

}

func CtrlCHandler(client *mongo.Client, logger *log.Logger) {

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		logger.Printf("- Ctrl+C pressed, exiting\n Signal recieved: %v\n", sig)
		if err := DisconnectClient(client, logger); err != nil {
			logger.Fatalf("Error disconnecting the client: %v\n", err)
		}
		os.Exit(0)
	}()
}
