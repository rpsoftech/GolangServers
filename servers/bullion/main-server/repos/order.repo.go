package bullion_main_server_repos

import (
	"errors"
	"fmt"
	"net/http"
	"time"

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
	OrderRepoStruct struct {
		collection *mongo.Collection
	}
)

const orderCollectionName = "Order"

var OrderRepo *OrderRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(orderCollectionName)
	OrderRepo = &OrderRepoStruct{
		collection: coll,
	}
	mongodb.AddUniqueIndexesToCollection([]string{"id"}, OrderRepo.collection)
	mongodb.AddIndexesToCollection([]string{"userId", "productGroupMapId", "groupId", "productId", "orderStatus", "createdAt"}, OrderRepo.collection)
}

func (repo *OrderRepoStruct) Save(entity *bullion_main_server_interfaces.OrderEntity) (*bullion_main_server_interfaces.OrderEntity, error) {
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

func (repo *OrderRepoStruct) BulkUpdate(entities *[]bullion_main_server_interfaces.OrderEntity) (*[]bullion_main_server_interfaces.OrderEntity, error) {
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

func (repo *OrderRepoStruct) findByFilter(filter *mongodb.MongoDbFilter) (*[]bullion_main_server_interfaces.OrderEntity, error) {
	var result []bullion_main_server_interfaces.OrderEntity
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

func (repo *OrderRepoStruct) FindOne(id string) (*bullion_main_server_interfaces.OrderEntity, error) {
	var result bullion_main_server_interfaces.OrderEntity

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

func (repo *OrderRepoStruct) DeleteOrderHistoryById(id string) error {
	_, err := repo.collection.DeleteOne(mongodb.MongoCtx, bson.D{{
		Key: "_id", Value: id,
	}})
	return err
}

func (repo *OrderRepoStruct) GetOrdersByBullionIdWithDateRangeAndOrderStatus(bullionId string, startDate time.Time, endDate time.Time, orderStatusArray *[]bullion_main_server_interfaces.OrderStatus) (*[]bullion_main_server_interfaces.OrderEntity, error) {
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Sort: &bson.D{{Key: "createdAt", Value: -1}},
		Conditions: &bson.D{
			{Key: "bullionId", Value: bullionId},
			{Key: "createdAt", Value: bson.D{{Key: "$gte", Value: startDate}, {Key: "$lte", Value: endDate}}},
			{Key: "orderStatus", Value: bson.D{{Key: "$in", Value: *orderStatusArray}}},
		},
	})
}

func (repo *OrderRepoStruct) GetUsersOrderPaginated(userId string, page int64, limit int64) (*[]bullion_main_server_interfaces.OrderEntity, error) {
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Sort: &bson.D{{Key: "createdAt", Value: -1}},
		Conditions: &bson.D{
			{Key: "userId", Value: userId},
		},
		Limit: limit,
		Skip:  page * limit,
	})
}

func (repo *OrderRepoStruct) GetUsersOrderPaginatedWithOrderStatusArray(userId string, orderStatusArray *[]bullion_main_server_interfaces.OrderStatus, page int64, limit int64) (*[]bullion_main_server_interfaces.OrderEntity, error) {
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Sort: &bson.D{{Key: "createdAt", Value: -1}},
		Conditions: &bson.D{
			{Key: "userId", Value: userId},
			{Key: "orderStatus", Value: bson.D{{Key: "$in", Value: *orderStatusArray}}},
		},
		Limit: limit,
		Skip:  page * limit,
	})
}
