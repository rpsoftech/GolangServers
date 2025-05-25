package jwelly_main_server_repos

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	jwelly_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/main-server/interfaces"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"github.com/rpsoftech/golang-servers/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type KeyValueDataRepoStruct struct {
	collection *mongo.Collection
	redis      *redis.RedisClientStruct
}

const keyValueDataCollectionName = "KeyValue"

var KeyValueDataRepo *KeyValueDataRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(keyValueDataCollectionName)
	KeyValueDataRepo = &KeyValueDataRepoStruct{
		collection: coll,
		redis:      redis.InitRedisAndRedisClient(),
	}
	mongodb.AddUniqueIndexesToCollection([]string{"id", "key"}, KeyValueDataRepo.collection)
}

func (repo *KeyValueDataRepoStruct) Save(entity *jwelly_main_server_interfaces.KeyValueDataStruct) (*jwelly_main_server_interfaces.KeyValueDataStruct, error) {
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
		Key: "key", Value: entity.Key,
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
	// go redis.RedisClient.SetStringData(entity.Key, entity.Value, entity.ExpiresIn)
	return entity, err
}

func (repo *KeyValueDataRepoStruct) cacheDataToRedis(entity *jwelly_main_server_interfaces.KeyValueDataStruct) {
	go redis.CacheDataToRedis(repo.redis, entity, fmt.Sprintf("keyvalue/%s", entity.Key), redis.TimeToLive_OneDay)
}
func (repo *KeyValueDataRepoStruct) BulkUpdate(entities *[]jwelly_main_server_interfaces.KeyValueDataStruct) (*[]jwelly_main_server_interfaces.KeyValueDataStruct, error) {
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
			bson.D{{Key: "$set", Value: entity}})
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

func (repo *KeyValueDataRepoStruct) FindOneByKey(key string) (*jwelly_main_server_interfaces.KeyValueDataStruct, error) {
	result := new(jwelly_main_server_interfaces.KeyValueDataStruct)
	if redisData := repo.redis.GetStringData(fmt.Sprintf("keyvalue/%s", key)); redisData != "" {
		if err := json.Unmarshal([]byte(redisData), result); err == nil {
			result.RestoreTimeStamp()
			return result, err
		}
	}
	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "key", Value: key,
	}}).Decode(&result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("KeyValue Entity identified by key %s not found", key),
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

func (repo *KeyValueDataRepoStruct) FindOne(id string) (*jwelly_main_server_interfaces.KeyValueDataStruct, error) {
	var result jwelly_main_server_interfaces.KeyValueDataStruct

	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "id", Value: id,
	}}).Decode(&result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("KeyValue Entity identified by id %s not found", id),
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
