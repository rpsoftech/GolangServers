package bullion_main_server_repos

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	FeedsRepoStruct struct {
		collection *mongo.Collection
	}
)

const feedCollectionName = "Feed"

var FeedsRepo *FeedsRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(feedCollectionName)
	FeedsRepo = &FeedsRepoStruct{
		collection: coll,
	}
	mongodb.AddUniqueIndexesToCollection([]string{"id"}, FeedsRepo.collection)
	mongodb.AddIndexesToCollection([]string{"bullionId", "createdAt"}, FeedsRepo.collection)
}

func (repo *FeedsRepoStruct) Save(entity *bullion_main_server_interfaces.FeedsEntity) (*bullion_main_server_interfaces.FeedsEntity, error) {
	if err := utility_functions.ValidateStructAndReturnReqError(entity, &interfaces.RequestError{
		StatusCode: http.StatusBadRequest,
		Code:       interfaces.ERROR_INVALID_ENTITY,
		Message:    "",
		Name:       "ERROR_INVALID_ENTITY",
	}); err != nil {
		return entity, err
	}
	entity.Updated()
	err := repo.collection.FindOneAndUpdate(mongodb.MongoCtx, bson.D{{
		Key: "_id", Value: entity.ID,
	}}, bson.D{{Key: "$set", Value: entity}}, mongodb.FindOneAndUpdateOptions).Err()
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
	return entity, err
}

func (repo *FeedsRepoStruct) findByFilter(filter *mongodb.MongoDbFilter) (*[]bullion_main_server_interfaces.FeedsEntity, error) {
	var result []bullion_main_server_interfaces.FeedsEntity
	opt := options.Find()
	if filter.Sort != nil {
		opt.SetSort(filter.Sort)
	}
	if filter.Limit > 0 {
		opt.SetLimit(filter.Limit)
	}
	if filter.Skip > 0 {
		opt.SetSkip(filter.Skip)
	}
	cursor, err := repo.collection.Find(mongodb.MongoCtx, filter.Conditions, opt)
	if err == nil {
		err = cursor.All(mongodb.MongoCtx, &result)
	}
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("Feeds Entities filtered By %v not found", filter),
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
	return &result, err
}

func (repo *FeedsRepoStruct) GetAllByBullionId(bullionId string) (*[]bullion_main_server_interfaces.FeedsEntity, error) {
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Conditions: &bson.D{{Key: "bullionId", Value: bullionId}},
	})
}

func (repo *FeedsRepoStruct) DeleteById(id string) error {
	_, err := repo.collection.DeleteOne(mongodb.MongoCtx, bson.D{{
		Key: "id", Value: id,
	}})

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("Feeds Entity identified by id %s not found", id),
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
	return err
}

func (repo *FeedsRepoStruct) GetPaginatedFeedInDescendingOrder(bullionId string, page int64, limit int64) (*[]bullion_main_server_interfaces.FeedsEntity, error) {
	println(limit)
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Conditions: &bson.D{{Key: "bullionId", Value: bullionId}},
		Sort:       &bson.D{{Key: "createdAt", Value: -1}},
		Limit:      limit,
		Skip:       page * limit,
	})
}

func (repo *FeedsRepoStruct) FindOne(id string) (*bullion_main_server_interfaces.FeedsEntity, error) {
	var result bullion_main_server_interfaces.FeedsEntity

	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "id", Value: id,
	}}).Decode(&result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("Feeds Entity identified by id %s not found", id),
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
	return &result, err
}
