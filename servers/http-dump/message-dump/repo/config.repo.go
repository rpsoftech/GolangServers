package messagedump_repo

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rpsoftech/golang-servers/interfaces"
	messagedump_interfaces "github.com/rpsoftech/golang-servers/servers/http-dump/message-dump/interfaces"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MessageDumpConfigRepoStruct struct {
	collection *mongo.Collection
}

const messageDumpConfigCollectionName = "MessageDumpConfigs"

var messageDumpConfigRepo *MessageDumpConfigRepoStruct

func InitAndReturnMessageDumpConfigRepo() *MessageDumpConfigRepoStruct {
	if messageDumpConfigRepo != nil {
		return messageDumpConfigRepo
	}
	coll := mongodb.MongoDatabase.Collection(messageDumpConfigCollectionName)
	messageDumpConfigRepo = &MessageDumpConfigRepoStruct{
		collection: coll,
	}
	mongodb.AddUniqueIndexesToCollection([]string{"id"}, messageDumpConfigRepo.collection)
	return messageDumpConfigRepo
}

func (repo *MessageDumpConfigRepoStruct) FindOne(id string) (*messagedump_interfaces.MessageDumpServerConfig, error) {
	result := new(messagedump_interfaces.MessageDumpServerConfig)

	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "id", Value: id,
	}}).Decode(result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("Product Entity identified by id %s not found", id),
				Name:       "ENTITY_NOT_FOUND",
			}
		} else {
			err = &interfaces.RequestError{
				StatusCode: 500,
				Code:       interfaces.ERROR_INTERNAL_SERVER,
				Message:    fmt.Sprintf("Internal Server Error: %s", err.Error()),
				Name:       "INTERNAL_ERROR",
			}
		}
	}
	return result, err
}
