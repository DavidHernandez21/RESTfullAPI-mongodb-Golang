package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

const (
	setContentType              = "content-type"
	jsonType                    = "application/json"
	internalError               = "Internal Error"
	errorInserting              = "Error while inserting the person: %v \n%v\n"
	errorMarshalling            = "Error while marshalling the result: %v \n%v\n"
	errorExausting              = "Error while exausting the request body: %v\n"
	errorQuerying               = "Error while querying the collection: %v \n%v\n"
	errorWrittingResponse       = "Error while writing the error response: %v\n"
	errorretrievingFromCursor   = "Error retrieving results from cursor: %v\n"
	errorClosingCursor          = "Error closing cursor: %v\n"
	errorFindingAllDocuments    = "Error while finding all documents: %v"
	errorParsingID              = "Error while parsing the id: %v \n%v\n"
	errorDeletingDocument       = "Error while deleting a document: %v \n%v\n"
	errorFindingDocument        = "Error while finding a document: %v \n%v\n"
	errorWrittingClientResponse = "Error while writing the client response: %v\n"
	errorMarshallingPerson      = "Error marshalling a Person: %v"
	errorUpdatingPerson         = "Error updating a Person: %v\n%v\n"
	errorWrittingUpdate         = "Error while writing the no update operation response: %v\n"
	errorValidatingPerson       = "Error validating person: %v"
	errorMarshallingBody        = "Error while marshalling the request body: %v\n"
	noPersonFound               = "No Person was found with the name: %v"
	noIDFound                   = "No Person was found with the id: %v"
	noUpdateOperation           = "No update operation was done to document with id: %v"
)

func exaustRequestBody(r io.ReadCloser, log *log.Logger) {
	_, copyErr := io.Copy(io.Discard, r)
	if copyErr != nil {
		log.Printf(errorExausting, copyErr)
	}
}

func (c *EndpointHandler) CreatePersonEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("CreatePersonEndpoint", c.logger)

	// defer stop()

	response.Header().Set(setContentType, jsonType)

	person := request.Context().Value(keyProduct{}).(data.Person)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	result, err := c.collection.InsertOne(ctx, person)

	if err != nil {

		http.Error(response, internalError, http.StatusInternalServerError)
		c.logger.Printf(errorInserting, person, err)
		// exaust the request body
		exaustRequestBody(request.Body, c.logger)
		return
	}

	err = json.NewEncoder(response).Encode(result)

	if err != nil {

		http.Error(response, internalError, http.StatusInternalServerError)
		c.logger.Printf(errorMarshalling, result, err)
		exaustRequestBody(request.Body, c.logger)
		return
	}

	exaustRequestBody(request.Body, c.logger)

}

func (c *EndpointHandler) GetPersonByNameEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPersonByNameEndpoint", c.logger)

	// defer stop()

	response.Header().Set(setContentType, jsonType)

	const NAME_ENDPOINT = "NAME_ENDPOINT"
	name := mux.Vars(request)[os.Getenv(NAME_ENDPOINT)]

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	const regexPattern = `(?:\A|\s)(%v)(?:\s|\z)`
	pattern := fmt.Sprintf(regexPattern, name)

	const regexOptions = "i"
	regexValue := primitive.Regex{Pattern: pattern, Options: regexOptions}

	const (
		bsonKey  = "firstname"
		regexKey = "$regex"
	)
	cursor, err := c.collection.Find(ctx, bson.D{primitive.E{Key: bsonKey, Value: bson.D{primitive.E{Key: regexKey, Value: regexValue}}}})

	defer func() {
		if err := cursor.Close(ctx); err != nil {
			c.logger.Printf(errorClosingCursor, err)
		}
	}()

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorQuerying, name, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}

		return
	}

	var people data.People

	people, err = appendPersonFromCursor(cursor, people, ctx, response, c.logger)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorretrievingFromCursor, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		return
	}

	err = people.ToJSON(response)

	if err == data.ErrNotFound {
		c.logger.Printf(noPersonFound, name)
		_, err := response.Write([]byte(`{ "message": "No Person was found with the name: '` + name + `'" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		return
	}

	if err != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorMarshalling, people, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		return

	}

}

func (c *EndpointHandler) GetPersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPersonById", c.logger)

	// defer stop()

	response.Header().Set(setContentType, jsonType)
	const ID = "id"
	paramsId := mux.Vars(request)[ID]

	id, errId := primitive.ObjectIDFromHex(paramsId)

	if errId != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorParsingID, paramsId, errId)
		_, err := response.Write([]byte(`{ "message": "` + errId.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		return
	}

	var person data.Person

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	const bsonKey = "_id"
	err := c.collection.FindOne(ctx, bson.D{{
		Key: bsonKey, Value: id,
	}}).Decode(&person)

	if err == mongo.ErrNoDocuments {
		c.logger.Printf(noIDFound, paramsId)
		_, err := response.Write([]byte(`{ "message": "No Person was found with the id: ` + paramsId + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		return
	}

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorFindingDocument, paramsId, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		return
	}

	err = person.ToJSON(response)

	if err != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorMarshalling, person, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		return

	}
}

func (c *EndpointHandler) DeletePersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("DeletePersonByIdEndpoint", c.logger)

	// defer stop()

	response.Header().Set(setContentType, jsonType)
	const ID = "id"
	paramsId := mux.Vars(request)[ID]

	id, err := primitive.ObjectIDFromHex(paramsId)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorParsingID, paramsId, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		exaustRequestBody(request.Body, c.logger)
		return
	}

	// var person data.Person

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	const bsonKey = "_id"
	deleteResult, err := c.collection.DeleteOne(ctx, bson.D{{Key: bsonKey, Value: id}})

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorDeletingDocument, paramsId, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		exaustRequestBody(request.Body, c.logger)
		return
	}

	if deleteResult.DeletedCount == 0 {
		c.logger.Printf(noIDFound, paramsId)
		_, err := response.Write([]byte(`{ "message": "No Person was found with the id: ` + paramsId + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		exaustRequestBody(request.Body, c.logger)
		return
	}

	_, err = response.Write([]byte(`{ "message": "Person with id: ` + paramsId + ` was deleted" }`))
	if err != nil {
		c.logger.Printf(errorWrittingClientResponse, err)
	}
	exaustRequestBody(request.Body, c.logger)

}

func (c *EndpointHandler) UpdatePersonByIdEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("UpdatePersonByIdEndpoint", c.logger)

	// defer stop()

	response.Header().Set(setContentType, jsonType)
	paramsId := mux.Vars(request)["id"]

	id, err := primitive.ObjectIDFromHex(paramsId)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorParsingID, paramsId, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		_, copyErr := io.Copy(io.Discard, request.Body)
		if copyErr != nil {
			c.logger.Printf(errorExausting, copyErr)
		}
		return
	}

	// var person data.PersonUpdate

	person := request.Context().Value(keyProduct{}).(data.PersonUpdate)

	personJson, err := json.Marshal(person)
	// c.logger.Printf("%s\n", personJson)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorMarshallingPerson, err)
		_, err := response.Write([]byte(`{ "message": "Error processing the request" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		_, copyErr := io.Copy(io.Discard, request.Body)
		if copyErr != nil {
			c.logger.Printf(errorExausting, copyErr)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	const regexPattern = `[{}"]`
	reSquareBrackets := regexp.MustCompile(regexPattern)
	personRegex := reSquareBrackets.ReplaceAllString(string(personJson), "")
	reComma := regexp.MustCompile(`,`)
	const colon = ":"
	personRegex = reComma.ReplaceAllString(personRegex, colon)
	keyValueSliceToUpdate := strings.Split(personRegex, colon)
	// c.logger.Println(keyValueSliceToUpdate[0], keyValueSliceToUpdate[1])

	lenKeystoUpdate := len(keyValueSliceToUpdate)

	collection := c.collection
	var updateResultsSum int64
	const bsonCommand = "$set"
	for i := 0; i < lenKeystoUpdate; i = i + 2 {
		updateResult, err := collection.UpdateByID(
			ctx,
			id,
			bson.D{
				{Key: bsonCommand, Value: bson.D{{Key: keyValueSliceToUpdate[i], Value: keyValueSliceToUpdate[i+1]}}},
			},
		)

		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
			c.logger.Printf(errorUpdatingPerson, paramsId, err)
			_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
			if err != nil {
				c.logger.Printf(errorWrittingResponse, err)
			}
			return
		}
		updateResultsSum += updateResult.ModifiedCount

	}

	if updateResultsSum == 0 {
		c.logger.Printf(noUpdateOperation, paramsId)
		_, err := response.Write([]byte(`{ "message": "No update operation was done to document with id: ` + paramsId + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingUpdate, err)
		}
		_, copyErr := io.Copy(io.Discard, request.Body)
		if copyErr != nil {
			c.logger.Printf(errorExausting, copyErr)
		}
		return
	}

	_, err = response.Write([]byte(`{ "message": "Person with id: ` + paramsId + ` was updated" }`))
	if err != nil {
		c.logger.Printf("Error while writing the update response: %v\n", err)
	}
	_, copyErr := io.Copy(io.Discard, request.Body)
	if copyErr != nil {
		c.logger.Printf(errorExausting, copyErr)
	}

}

func (c *EndpointHandler) GetPeopleEndpoint(response http.ResponseWriter, request *http.Request) {

	// stop := timer.StartTimer("GetPeopleEndpoint", c.logger)

	// defer stop()

	response.Header().Set(setContentType, jsonType)

	var people data.People

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()

	cursor, err := c.collection.Find(ctx, bson.M{})

	defer func() {
		if err := cursor.Close(ctx); err != nil {
			c.logger.Printf(errorClosingCursor, err)
		}
	}()

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorFindingAllDocuments, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}

		return
	}

	people, err = appendPersonFromCursor(cursor, people, ctx, response, c.logger)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf("Error while appending people from cursor: %v", err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
		}
		return
	}

	err = people.ToJSON(response)

	if err != nil {

		response.WriteHeader(http.StatusInternalServerError)
		c.logger.Printf(errorMarshalling, people, err)
		_, err := response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		if err != nil {
			c.logger.Printf(errorWrittingResponse, err)
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
			c.logger.Printf(errorMarshallingBody, err)
			return
		}

		if err := person.Validate(); err != nil {
			c.logger.Printf(errorValidatingPerson, err)
			http.Error(response, fmt.Sprintf(errorValidatingPerson, err), http.StatusBadRequest)
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
			c.logger.Printf(errorMarshallingBody, err)
			return
		}

		if err := person.Validate(); err != nil {
			c.logger.Printf(errorValidatingPerson, err)
			http.Error(response, fmt.Sprintf(errorValidatingPerson, err), http.StatusBadRequest)
			return
		}

		// add the product to the context
		ctx := context.WithValue(request.Context(), keyProduct{}, person)
		request = request.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(response, request)
	})
}
