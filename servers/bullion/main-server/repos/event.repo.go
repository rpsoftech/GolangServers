package bullion_main_server_repos

import (
	"errors"
	"fmt"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/events"
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
)

type EventRepoStruct struct {
	collection *mongo.Collection
}

const eventsCollectionName = "Events"

var EventRepo *EventRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(eventsCollectionName)
	EventRepo = &EventRepoStruct{
		collection: coll,
	}
	addUniqueIndexesToCollection([]string{"id"}, EventRepo.collection)
	addIndexesToCollection([]string{"key", "bullionId", "occurredAt", "eventName", "parentNames"}, EventRepo.collection)
}

func (repo *EventRepoStruct) Save(entity *events.BaseEvent) error {
	_, err := repo.collection.InsertOne(mongodb.MongoCtx, entity)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			err = &interfaces.RequestError{
				StatusCode: 500,
				Code:       interfaces.ERROR_INTERNAL_SERVER,
				Message:    fmt.Sprintf("Internal Server Error: %s", err.Error()),
				Name:       "INTERNAL_ERROR",
			}
		} else {
			err = nil
		}
	}
	return err
}
func (repo *EventRepoStruct) SaveAll(entity *[]any) error {
	_, err := repo.collection.InsertMany(mongodb.MongoCtx, *entity)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			err = &interfaces.RequestError{
				StatusCode: 500,
				Code:       interfaces.ERROR_INTERNAL_SERVER,
				Message:    fmt.Sprintf("Internal Server Error: %s", err.Error()),
				Name:       "INTERNAL_ERROR",
			}
		} else {
			err = nil
		}
	}
	return err
}
