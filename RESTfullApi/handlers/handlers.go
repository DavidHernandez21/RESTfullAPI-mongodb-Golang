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

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	result, err := c.collection.InsertOne(ctx, person)

	if err != nil {

		http.Error(response, "Internal Error", http.StatusInternalServerError)
		c.logger.Printf("Error while inserting the person: %v \n%v\n", person, err)
		return
	}

	err = json.NewEncoder(response).Encode(result)

	if err != nil {

		http.Error(response, "Internal Error", http.StatusInternalServerError)
		c.logger.Printf("Error while marshalling the result: %v \n%v\n", result, err)
		return
	}

}

func (c *EndpointHandler) GetPersonByNameEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPersonByNameEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")

	name := mux.Vars(request)[os.Getenv("NAME_ENDPOINT")]

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	pattern := fmt.Sprintf(`(?:\A|\s)(%v)(?:\s|\z)`, name)

	regexValue := primitive.Regex{Pattern: pattern, Options: "i"}

	cursor, err := c.collection.Find(ctx, bson.D{primitive.E{Key: "firstname", Value: bson.D{primitive.E{Key: "$regex", Value: regexValue}}}})

	defer func() {
		if err := cursor.Close(ctx); err != nil {
			c.logger.Printf("Error closing cursor: %v\n", err)
		}
	}()

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while querying the collection: %v \n%v\n", name, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}

		return
	}

	var people data.People

	people, err = appendPersonFromCursor(cursor, people, ctx, response, c.logger)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error retrieving results from cursor: %v\n", err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	err = people.ToJSON(response)

	if err == data.ErrNotFound {
		c.logger.Printf("No Person was found with the name: %v", name)
		_, err := response.Write([]byte(`{ "message": "No Person was found with the name: '` + name + `'" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	if err != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while marshalling the result: %v \n%v\n", people, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return

	}

}

func (c *EndpointHandler) GetPersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPersonById", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")
	paramsId := mux.Vars(request)["id"]

	id, errId := primitive.ObjectIDFromHex(paramsId)

	if errId != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while parsing the id: %v \n%v\n", paramsId, errId)
		_, err := response.Write([]byte(`{ "message": "` + errId.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	var person data.Person

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	err := c.collection.FindOne(ctx, bson.D{{
		Key: "_id", Value: id,
	}}).Decode(&person)

	if err == mongo.ErrNoDocuments {
		c.logger.Printf("No Person was found with the id: %v", paramsId)
		_, err := response.Write([]byte(`{ "message": "No Person was found with the id: ` + paramsId + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while finding a document: %v \n%v\n", paramsId, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	err = person.ToJSON(response)

	if err != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while marshalling the result: %v \n%v\n", person, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return

	}
}

func (c *EndpointHandler) DeletePersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("DeletePersonByIdEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")
	paramsId := mux.Vars(request)["id"]

	id, err := primitive.ObjectIDFromHex(paramsId)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while parsing the id: %v \n%v\n", paramsId, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	// var person data.Person

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	deleteResult, err := c.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while deleting a document: %v \n%v\n", paramsId, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	if deleteResult.DeletedCount == 0 {
		c.logger.Printf("No Person was found with the id: %v", paramsId)
		_, err := response.Write([]byte(`{ "message": "No Person was found with the id: ` + paramsId + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	_, err = response.Write([]byte(`{ "message": "Person with id: ` + paramsId + ` was deleted" }`))
	if err != nil {
		c.logger.Printf("Error while writing the client response: %v\n", err)
	}

}

func (c *EndpointHandler) UpdatePersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("UpdatePersonByIdEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")
	paramsId := mux.Vars(request)["id"]

	id, err := primitive.ObjectIDFromHex(paramsId)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while parsing the id: %v \n%v\n", paramsId, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	// var person data.PersonUpdate

	person := request.Context().Value(keyProduct{}).(data.PersonUpdate)

	personJson, err := json.Marshal(person)
	// c.logger.Printf("%s\n", personJson)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error marshalling a Person: %v", err)
		_, err := response.Write([]byte(`{ "message": "Error processing the request" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	reSquareBrackets := regexp.MustCompile(`[{}"]`)
	personRegex := reSquareBrackets.ReplaceAllString(string(personJson), "")
	reComma := regexp.MustCompile(`,`)
	personRegex = reComma.ReplaceAllString(personRegex, ":")
	keyValueSliceToUpdate := strings.Split(personRegex, ":")
	// c.logger.Println(keyValueSliceToUpdate[0], keyValueSliceToUpdate[1])

	lenKeystoUpdate := len(keyValueSliceToUpdate)

	collection := c.collection
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
			_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
			if err != nil {
				c.logger.Printf("Error while writing the error response: %v\n", err)
			}
			return
		}
		updateResultsSum += updateResult.ModifiedCount

	}

	if updateResultsSum == 0 {
		c.logger.Printf("No update operation was done to document with id: %v", paramsId)
		_, err := response.Write([]byte(`{ "message": "No update operation was done to document with id: ` + paramsId + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the no update operation response: %v\n", err)
		}
		return
	}

	_, err = response.Write([]byte(`{ "message": "Person with id: ` + paramsId + ` was updated" }`))
	if err != nil {
		c.logger.Printf("Error while writing the update response: %v\n", err)
	}

}

func (c *EndpointHandler) GetPeopleEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPeopleEndpoint", c.logger)

	// defer stop()

	response.Header().Set("content-type", "application/json")

	var people data.People

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	cursor, err := c.collection.Find(ctx, bson.M{})

	defer func() {
		if err := cursor.Close(ctx); err != nil {
			c.logger.Printf("Error closing cursor: %v\n", err)
		}
	}()

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while finding all documents: %v", err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}

		return
	}

	people, err = appendPersonFromCursor(cursor, people, ctx, response, c.logger)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while appending people from cursor: %v", err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
		return
	}

	err = people.ToJSON(response)

	if err != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while marshalling the result: %v \n%v\n", people, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf("Error while writing the error response: %v\n", err)
		}
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
		err := cursor.Decode(&person)
		if err != nil {
			logger.Printf("Error decoding person: %v", err)
			continue
		}
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

		if err := person.Validate(); err != nil {
			c.logger.Printf("Error validating person: %v", err)
			http.Error(response, fmt.Sprintf("Error validating person: %v", err), http.StatusBadRequest)
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

		if err := person.Validate(); err != nil {
			c.logger.Printf("Error validating person: %v", err)
			http.Error(response, fmt.Sprintf("Error validating person: %v", err), http.StatusBadRequest)
			return
		}

		// add the product to the context
		ctx := context.WithValue(request.Context(), keyProduct{}, person)
		request = request.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(response, request)
	})
}
