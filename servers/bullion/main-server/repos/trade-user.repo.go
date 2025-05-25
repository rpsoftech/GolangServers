package bullion_main_server_repos

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"github.com/rpsoftech/golang-servers/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	TradeUserRepoStruct struct {
		collection *mongo.Collection
		redis      *redis.RedisClientStruct
	}
)

const tradeUserCollectionName = "TradeUser"

var TradeUserRepo *TradeUserRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(tradeUserCollectionName)
	TradeUserRepo = &TradeUserRepoStruct{
		collection: coll,
		redis:      redis.InitRedisAndRedisClient(),
	}
	mongodb.AddUniqueIndexesToCollection([]string{"id"}, TradeUserRepo.collection)
	mongodb.AddIndexesToCollection([]string{"bullionId", "isActive"}, TradeUserRepo.collection)
	mongodb.AddComboUniqueIndexesToCollection([]string{"email", "bullionId"}, TradeUserRepo.collection)
	mongodb.AddComboUniqueIndexesToCollection([]string{"number", "bullionId"}, TradeUserRepo.collection)
	mongodb.AddComboUniqueIndexesToCollection([]string{"uNumber", "bullionId"}, TradeUserRepo.collection)
	mongodb.AddComboUniqueIndexesToCollection([]string{"userName", "bullionId"}, TradeUserRepo.collection)
}

func (repo *TradeUserRepoStruct) Save(entity *bullion_main_server_interfaces.TradeUserEntity) (*bullion_main_server_interfaces.TradeUserEntity, error) {
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
	repo.cacheDataToRedis(entity)
	return entity, err
}

func (repo *TradeUserRepoStruct) FindDuplicateUser(email string, number string, userName string, bullionId string) (*[]bullion_main_server_interfaces.TradeUserEntity, error) {
	condition := bson.D{
		{
			Key: "$and",
			Value: bson.A{
				bson.D{{Key: "bullionId", Value: bullionId}},
			},
		},
		{
			Key: "$or",
			Value: bson.A{
				bson.D{{Key: "email", Value: email}},
				bson.D{{Key: "number", Value: number}},
				bson.D{{Key: "userName", Value: userName}},
			},
		},
	}
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Conditions: &condition,
	})
}

func (repo *TradeUserRepoStruct) findByFilter(filter *mongodb.MongoDbFilter) (*[]bullion_main_server_interfaces.TradeUserEntity, error) {
	var result []bullion_main_server_interfaces.TradeUserEntity
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

func (repo *TradeUserRepoStruct) FindAllInActiveUser(bullionId string) (*[]bullion_main_server_interfaces.TradeUserEntity, error) {
	return repo.findByFilter(&mongodb.MongoDbFilter{
		Conditions: &bson.D{
			{
				Key: "$and",
				Value: bson.A{
					bson.D{{Key: "bullionId", Value: bullionId}},
					bson.D{{Key: "isActive", Value: false}},
				},
			},
		},
	})
}

func (repo *TradeUserRepoStruct) FindOne(id string) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	result := new(bullion_main_server_interfaces.TradeUserEntity)
	if redisData := repo.redis.GetStringData(fmt.Sprintf("tradeUser/%s", id)); redisData != "" {
		if err := json.Unmarshal([]byte(redisData), result); err == nil {
			result.RestoreTimeStamp()
			return result, err
		}
	}
	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "id", Value: id,
	}}).Decode(result)

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
	repo.cacheDataToRedis(result)
	return result, err
}

func (repo *TradeUserRepoStruct) cacheDataToRedis(entity *bullion_main_server_interfaces.TradeUserEntity) {
	go redis.CacheDataToRedis(repo.redis, entity, fmt.Sprintf("tradeUser/%s", entity.ID), redis.TimeToLive_OneDay)
}

func (repo *TradeUserRepoStruct) findOneByCondition(bullionId string, condition *bson.E) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	var result bullion_main_server_interfaces.TradeUserEntity

	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{
		{Key: "bullionId", Value: bullionId},
		*condition,
	}).Decode(&result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("Feeds Entity identified by %s %s not found", condition.Key, condition.Value),
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

func (repo *TradeUserRepoStruct) FindOneByEmail(bullionId string, email string) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	return repo.findOneByCondition(bullionId, &bson.E{Key: "email", Value: email})
}

func (repo *TradeUserRepoStruct) FindOneByNumber(bullionId string, number string) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	return repo.findOneByCondition(bullionId, &bson.E{Key: "number", Value: number})
}

func (repo *TradeUserRepoStruct) FindOneByUNumber(bullionId string, uNumber string) (*bullion_main_server_interfaces.TradeUserEntity, error) {
	return repo.findOneByCondition(bullionId, &bson.E{Key: "uNumber", Value: uNumber})
}
