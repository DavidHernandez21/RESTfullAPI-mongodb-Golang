package clients

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
		// return err
	}

	if os.Getenv("MONGODB_URI_WO_DATABASE") == "" {
		return errors.New("mongo URI is not set")
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

func DisconnectClient(ctx context.Context, client *mongo.Client, logger *log.Logger) error {

	stop := timer.StartTimer("DisconnectClient", logger)

	defer stop()

	if err := client.Disconnect(ctx); err != nil {
		return err
	}

	logger.Println("mongo client disconnected")

	return nil

}

func CtrlCHandler(ctx context.Context, client *mongo.Client, logger *log.Logger, server *http.Server) {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	errchan := make(chan error, 1)
	go func() {

		sig := <-c
		logger.Printf("- Ctrl+C pressed, exiting\n Signal recieved: %v\n", sig)
		if err := DisconnectClient(ctx, client, logger); err != nil {
			logger.Printf("Error disconnecting the client: %v\n", err)
			errchan <- err
		}
		if err := server.Shutdown(ctx); err != nil {
			logger.Printf("Error shutting down the server: %v\n", err)
			errchan <- err
		}
		close(errchan)

	}()

	var wg sync.WaitGroup
	wg.Add(1)
	errCheck := false
	go func() {

		for err := range errchan {
			logger.Printf("Error: %v\n", err)
			errCheck = true
		}
		wg.Done()

	}()

	go func() {
		wg.Wait()

		if errCheck {
			os.Exit(1)
		}
		logger.Println("Exiting")
		os.Exit(0)
	}()
}
