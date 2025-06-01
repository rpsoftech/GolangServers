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
	ProductGroupMapRepoStruct struct {
		collection *mongo.Collection
	}
)

const productGroupMapCollectionName = "ProductGroupMap"

var ProductGroupMapRepo *ProductGroupMapRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(productGroupMapCollectionName)
	ProductGroupMapRepo = &ProductGroupMapRepoStruct{
		collection: coll,
	}

	mongodb.AddUniqueIndexesToCollection([]string{"id"}, ProductGroupMapRepo.collection)
	mongodb.AddComboUniqueIndexesToCollection([]string{"groupId", "productId"}, ProductGroupMapRepo.collection)
	mongodb.AddComboIndexesToCollection([]string{"bullionId", "groupId"}, ProductGroupMapRepo.collection)
	mongodb.AddIndexesToCollection([]string{"bullionId", "createdAt", "groupId"}, ProductGroupMapRepo.collection)
}

func (repo *ProductGroupMapRepoStruct) Save(entity *bullion_main_server_interfaces.TradeUserGroupMapEntity) (*bullion_main_server_interfaces.TradeUserGroupMapEntity, error) {
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

func (repo *ProductGroupMapRepoStruct) BulkUpdate(entities *[]bullion_main_server_interfaces.TradeUserGroupMapEntity) (*[]bullion_main_server_interfaces.TradeUserGroupMapEntity, error) {
	models := make([]mongo.WriteModel, len(*entities))
	for i, entity := range *entities {
		if err := utility_functions.ValidateStructAndReturnReqError(entity, &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_ENTITY,
			Message:    "",
			Name:       "ERROR_INVALID_ENTITY",
		}); err != nil {
			return nil, err
		}
		entity.Updated()
		models[i] = mongo.NewUpdateOneModel().SetFilter(bson.D{{Key: "_id", Value: entity.ID}}).SetUpdate(
			bson.D{{Key: "$set", Value: entity}}).SetUpsert(true)
	}
	_, err := repo.collection.BulkWrite(mongodb.MongoCtx, models)
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
	return entities, err
}

func (repo *ProductGroupMapRepoStruct) findByFilter(filter *mongodb.MongoDbFilter) (*[]bullion_main_server_interfaces.TradeUserGroupMapEntity, error) {
	var result []bullion_main_server_interfaces.TradeUserGroupMapEntity
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

func (repo *ProductGroupMapRepoStruct) GetAllByBullionId(bullionId string) (*[]bullion_main_server_interfaces.TradeUserGroupMapEntity, error) {
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Conditions: &bson.D{{Key: "bullionId", Value: bullionId}},
	})
}
func (repo *ProductGroupMapRepoStruct) GetAllByGroupId(groupId string, bullionId string) (*[]bullion_main_server_interfaces.TradeUserGroupMapEntity, error) {
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Conditions: &bson.D{
			{Key: "groupId", Value: groupId},
			{Key: "bullionId", Value: bullionId},
		},
	})
}

func (repo *ProductGroupMapRepoStruct) FindOne(id string) (*bullion_main_server_interfaces.TradeUserGroupMapEntity, error) {
	var result bullion_main_server_interfaces.TradeUserGroupMapEntity
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
