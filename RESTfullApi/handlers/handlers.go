package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"RESTfullAPi-Golang/RESTfullApi/data"
	"RESTfullAPi-Golang/RESTfullApi/timer"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	CreatePersonEndpoint struct {
		Logger     *log.Logger
		Collection *mongo.Collection
	}

	GetPersonByIdEndpoint struct {
		Logger     *log.Logger
		Collection *mongo.Collection
	}

	GetPeopleEndpoint struct {
		Logger     *log.Logger
		Collection *mongo.Collection
	}

	GetPersonByNameEndpoint struct {
		Logger     *log.Logger
		Collection *mongo.Collection
	}
)

func (c *CreatePersonEndpoint) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	stop := timer.StartTimer("CreatePersonEndpoint", c.Logger)

	defer stop()

	response.Header().Set("content-type", "application/json")
	var person data.Person

	err := person.FromJSON(request.Body)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.Logger.Printf("Error while marshalling the request body: %v\n", err)
		return
	}

	collection := c.Collection

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err1 := collection.InsertOne(ctx, person)

	if err1 != nil {

		http.Error(response, "Internal Error", http.StatusInternalServerError)
		c.Logger.Printf("Error while inserting the person: %v \n%v\n", person, err1)
		return
	}

	err2 := json.NewEncoder(response).Encode(result)

	if err2 != nil {

		http.Error(response, "Internal Error", http.StatusInternalServerError)
		c.Logger.Printf("Error while marshalling the result: %v \n%v\n", result, err2)
		return
	}

}

func (c *GetPersonByNameEndpoint) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	stop := timer.StartTimer("GetPersonByNameEndpoint", c.Logger)

	defer stop()

	response.Header().Set("content-type", "application/json")

	name := mux.Vars(request)["name"]

	collection := c.Collection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	pattern := fmt.Sprintf("^%v.*", name)

	regexValue := primitive.Regex{Pattern: pattern, Options: "i"}

	cursor, err := collection.Find(ctx, bson.D{primitive.E{Key: "firstname", Value: bson.D{primitive.E{Key: "$regex", Value: regexValue}}}})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))

		return
	}
	defer cursor.Close(ctx)

	var people data.People

	people = appendPersonFromCursor(cursor, people, ctx, response, c.Logger)

	err1 := people.ToJSON(response)

	if err1 != nil {

		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err1.Error() + `" }`))
		return

	}

}

func (c *GetPersonByIdEndpoint) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	stop := timer.StartTimer("GetPersonByIdEndpoint", c.Logger)

	defer stop()

	response.Header().Set("content-type", "application/json")
	paramsId := mux.Vars(request)["id"]

	id, errId := primitive.ObjectIDFromHex(paramsId)

	if errId != nil {
		response.WriteHeader(http.StatusInternalServerError)

		response.Write([]byte(`{ "message": "` + errId.Error() + `" }`))
		return
	}

	var person data.Person

	collection := c.Collection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err := collection.FindOne(ctx, data.Person{ID: id}).Decode(&person)

	if err == mongo.ErrNoDocuments {
		c.Logger.Printf("No Person was found with the id: %v", paramsId)
		response.Write([]byte(`{ "message": "No Person was found with the id: ` + paramsId + `" }`))
		return
	}

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)

		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}

	err1 := person.ToJSON(response)

	if err1 != nil {

		response.WriteHeader(http.StatusInternalServerError)

		response.Write([]byte(`{ "message": "` + err1.Error() + `" }`))
		return

	}
}

func (c *GetPeopleEndpoint) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	stop := timer.StartTimer("GetPeopleEndpoint", c.Logger)

	defer stop()

	response.Header().Set("content-type", "application/json")

	var people data.People

	collection := c.Collection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))

		return
	}
	defer cursor.Close(ctx)

	people = appendPersonFromCursor(cursor, people, ctx, response, c.Logger)

	err2 := people.ToJSON(response)

	if err2 != nil {

		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err2.Error() + `" }`))
		return

	}

}

func NewGetPersonByIdEndpoint(logger *log.Logger, collection *mongo.Collection) *GetPersonByIdEndpoint {
	return &GetPersonByIdEndpoint{
		Logger:     logger,
		Collection: collection,
	}
}

func NewCreatePersonEndpoint(logger *log.Logger, collection *mongo.Collection) *CreatePersonEndpoint {
	return &CreatePersonEndpoint{
		Logger:     logger,
		Collection: collection,
	}
}

func NewGetPeopleEndpoint(logger *log.Logger, collection *mongo.Collection) *GetPeopleEndpoint {
	return &GetPeopleEndpoint{
		Logger:     logger,
		Collection: collection,
	}
}

func NewGetPersonByNameEndpoint(logger *log.Logger, collection *mongo.Collection) *GetPersonByNameEndpoint {
	return &GetPersonByNameEndpoint{
		Logger:     logger,
		Collection: collection,
	}
}

func appendPersonFromCursor(cursor *mongo.Cursor, people data.People, ctx context.Context, response http.ResponseWriter, logger *log.Logger) data.People {

	stop := timer.StartTimer("appendPersonFromCursor", logger)

	defer stop()

	for cursor.Next(ctx) {

		var person data.Person
		cursor.Decode(&person)
		people = append(people, &person)
	}
	if err := cursor.Err(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return nil
	}

	return people

}
