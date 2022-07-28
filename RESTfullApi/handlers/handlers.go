package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github/DavidHernandez21/RESTfullAPi-Golang/RESTfullApi/data"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	EndpointHandler struct {
		logger     *log.Logger
		collection *mongo.Collection
		timeout    time.Duration
	}

	keyProduct struct{}

	option func(handler *EndpointHandler)
)

func (c *EndpointHandler) CreatePersonEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("CreatePersonEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")

	person := request.Context().Value(keyProduct{}).(data.Person)

	collection := c.collection

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	result, err2 := collection.InsertOne(ctx, person)

	if err2 != nil {

		http.Error(response, "Internal Error", http.StatusInternalServerError)
		c.logger.Printf("Error while inserting the person: %v \n%v\n", person, err2)
		return
	}

	err3 := json.NewEncoder(response).Encode(result)

	if err3 != nil {

		http.Error(response, "Internal Error", http.StatusInternalServerError)
		c.logger.Printf("Error while marshalling the result: %v \n%v\n", result, err3)
		return
	}

}

func (c *EndpointHandler) GetPersonByNameEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPersonByNameEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")

	name := mux.Vars(request)[os.Getenv("NAME_ENDPOINT")]

	collection := c.collection
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	pattern := fmt.Sprintf(`(?:\A|\s)(%v)(?:\s|\z)`, name)

	regexValue := primitive.Regex{Pattern: pattern, Options: "i"}

	cursor, err := collection.Find(ctx, bson.D{primitive.E{Key: "firstname", Value: bson.D{primitive.E{Key: "$regex", Value: regexValue}}}})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while querying the collection: %v \n%v\n", name, err)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))

		return
	}
	defer cursor.Close(ctx)

	var people data.People

	people, err = appendPersonFromCursor(cursor, people, ctx, response, c.logger)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error retrieving results from cursor: %v\n", err)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}

	err1 := people.ToJSON(response)

	if err1 == data.ErrNotFound {
		c.logger.Printf("No Person was found with the name: %v", name)
		response.Write([]byte(`{ "message": "No Person was found with the name: '` + name + `'" }`))
		return
	}

	if err1 != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while marshalling the result: %v \n%v\n", people, err1)
		response.Write([]byte(`{ "message": "` + err1.Error() + `" }`))
		return

	}

}

func (c *EndpointHandler) GetPersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPersonByIdEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")
	paramsId := mux.Vars(request)["id"]

	id, errId := primitive.ObjectIDFromHex(paramsId)

	if errId != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while parsing the id: %v \n%v\n", paramsId, errId)
		response.Write([]byte(`{ "message": "` + errId.Error() + `" }`))
		return
	}

	var person data.Person

	collection := c.collection
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	err := collection.FindOne(ctx, bson.D{{
		Key: "_id", Value: id,
	}}).Decode(&person)

	if err == mongo.ErrNoDocuments {
		c.logger.Printf("No Person was found with the id: %v", paramsId)
		response.Write([]byte(`{ "message": "No Person was found with the id: ` + paramsId + `" }`))
		return
	}

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while finding a document: %v \n%v\n", paramsId, err)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}

	err1 := person.ToJSON(response)

	if err1 != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while marshalling the result: %v \n%v\n", person, err1)
		response.Write([]byte(`{ "message": "` + err1.Error() + `" }`))
		return

	}
}

func (c *EndpointHandler) DeletePersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("DeletePersonByIdEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")
	paramsId := mux.Vars(request)["id"]

	id, errId := primitive.ObjectIDFromHex(paramsId)

	if errId != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while parsing the id: %v \n%v\n", paramsId, errId)
		response.Write([]byte(`{ "message": "` + errId.Error() + `" }`))
		return
	}

	// var person data.Person

	collection := c.collection
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	deleteResult, err := collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while deleting a document: %v \n%v\n", paramsId, err)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}

	if deleteResult.DeletedCount == 0 {
		c.logger.Printf("No Person was found with the id: %v", paramsId)
		response.Write([]byte(`{ "message": "No Person was found with the id: ` + paramsId + `" }`))
		return
	}

	response.Write([]byte(`{ "message": "Person with id: ` + paramsId + ` was deleted" }`))

}

func (c *EndpointHandler) UpdatePersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("UpdatePersonByIdEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")
	paramsId := mux.Vars(request)["id"]

	id, errId := primitive.ObjectIDFromHex(paramsId)

	if errId != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while parsing the id: %v \n%v\n", paramsId, errId)
		response.Write([]byte(`{ "message": "` + errId.Error() + `" }`))
		return
	}

	// var person data.PersonUpdate

	person := request.Context().Value(keyProduct{}).(data.PersonUpdate)

	personJson, errMarshalling := json.Marshal(person)
	// c.logger.Printf("%s\n", personJson)

	if errMarshalling != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error marshalling a Person: %v", errMarshalling)
		response.Write([]byte(`{ "message": "Error processing the request" }`))
	}

	collection := c.collection
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	personJsonString := string(personJson)
	reSquareBrackets := regexp.MustCompile(`[{}"]`)
	personRegex := reSquareBrackets.ReplaceAllString(personJsonString, "")
	reComma := regexp.MustCompile(`,`)
	personRegex = reComma.ReplaceAllString(personRegex, ":")
	keyValueSliceToUpdate := strings.Split(personRegex, ":")
	// c.logger.Println(keyValueSliceToUpdate[0], keyValueSliceToUpdate[1])

	lenKeystoUpdate := len(keyValueSliceToUpdate)

	var updateResultsSum int64
	for i := 0; i < lenKeystoUpdate; i = i + 2 {
		updateResult, err := collection.UpdateByID(
			ctx,
			id,
			bson.D{
				{Key: "$set", Value: bson.D{{Key: keyValueSliceToUpdate[i], Value: keyValueSliceToUpdate[i+1]}}},
			},
		)

		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
			c.logger.Printf("Error updating a Person: %v\n%v\n", paramsId, err)
			response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
			return
		}
		updateResultsSum += updateResult.ModifiedCount

	}

	if updateResultsSum == 0 {
		c.logger.Printf("No update operation was done to document with id: %v", paramsId)
		response.Write([]byte(`{ "message": "No update operation was done to document with id: ` + paramsId + `" }`))
		return
	}

	response.Write([]byte(`{ "message": "Person with id: ` + paramsId + ` was updated" }`))

}

func (c *EndpointHandler) GetPeopleEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPeopleEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")

	var people data.People

	collection := c.collection
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while finding all documents: %v", err)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))

		return
	}
	defer cursor.Close(ctx)

	people, err = appendPersonFromCursor(cursor, people, ctx, response, c.logger)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while appending people from cursor: %v", err)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}

	err2 := people.ToJSON(response)

	if err2 != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while marshalling the result: %v \n%v\n", people, err2)
		response.Write([]byte(`{ "message": "` + err2.Error() + `" }`))
		return

	}

}

func NewEndpointHandler(logger *log.Logger, collection *mongo.Collection, opts ...option) *EndpointHandler {
	handler := &EndpointHandler{
		logger:     logger,
		collection: collection,
		timeout:    5 * time.Second,
	}

	for i := range opts {
		opts[i](handler)

	}

	return handler

}

func appendPersonFromCursor(cursor *mongo.Cursor, people data.People, ctx context.Context, response http.ResponseWriter, logger *log.Logger) (data.People, error) {

	// stop := timer.StartTimer("appendPersonFromCursor", logger)

	// defer stop()

	for cursor.Next(ctx) {

		var person data.Person
		cursor.Decode(&person)
		people = append(people, &person)
	}
	if err := cursor.Err(); err != nil {

		return nil, err
	}

	return people, nil

}

func WithTimeout(timeout time.Duration) option {
	return func(handler *EndpointHandler) {
		handler.timeout = timeout
	}
}

func (c *EndpointHandler) MiddlewareValidateProduct(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var person data.Person

		err := person.FromJSON(request.Body)

		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
			c.logger.Printf("Error while marshalling the request body: %v\n", err)
			return
		}

		if err1 := person.Validate(); err1 != nil {
			c.logger.Printf("Error validating person: %v", err1)
			http.Error(response, fmt.Sprintf("Error validating person: %v", err1), http.StatusBadRequest)
			return
		}

		// add the product to the context
		ctx := context.WithValue(request.Context(), keyProduct{}, person)
		request = request.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(response, request)
	})
}

func (c *EndpointHandler) MiddlewareValidateUpdateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var person data.PersonUpdate

		err := person.FromJSON(request.Body)

		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
			c.logger.Printf("Error while marshalling the request body: %v\n", err)
			return
		}

		if err1 := person.Validate(); err1 != nil {
			c.logger.Printf("Error validating person: %v", err1)
			http.Error(response, fmt.Sprintf("Error validating person: %v", err1), http.StatusBadRequest)
			return
		}

		// add the product to the context
		ctx := context.WithValue(request.Context(), keyProduct{}, person)
		request = request.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(response, request)
	})
}
