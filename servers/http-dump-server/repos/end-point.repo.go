package dump_server_repos

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	dump_server_env "github.com/rpsoftech/golang-servers/servers/http-dump-server/env"
	dump_server_interfaces "github.com/rpsoftech/golang-servers/servers/http-dump-server/interfaces"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"github.com/rpsoftech/golang-servers/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type EndPointRepoStruct struct {
	collection *mongo.Collection
	redis      *redis.RedisClientStruct
}

const endPointCollectionName = "EndPoint"

var EndPointRepo *EndPointRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(endPointCollectionName)
	EndPointRepo = &EndPointRepoStruct{
		collection: coll,
		redis:      redis.InitRedisAndRedisClient(),
	}
	mongodb.AddUniqueIndexesToCollection([]string{"id"}, EndPointRepo.collection)
	// mongodb.AddComboUniqueIndexesToCollection([]string{"userName", "bullionId"}, EndPointRepo.collection)
}
func (repo *EndPointRepoStruct) cacheDataToRedis(entity *dump_server_interfaces.EndPoint) {
	go redis.CacheDataToRedis(repo.redis, &entity, dump_server_env.GetRedisKey(entity.ID), redis.TimeToLive_OneDay)
}

func (repo *EndPointRepoStruct) Save(entity *dump_server_interfaces.EndPoint) (*dump_server_interfaces.EndPoint, error) {
	var result dump_server_interfaces.EndPoint
	err := repo.collection.FindOneAndUpdate(mongodb.MongoCtx, bson.D{{
		Key: "_id", Value: entity.ID,
	}}, bson.D{{Key: "$set", Value: entity}}, mongodb.FindOneAndUpdateOptions).Decode(&result)
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
	return &result, err
}

func (repo *EndPointRepoStruct) FindOne(id string) (*dump_server_interfaces.EndPoint, error) {
	result := new(dump_server_interfaces.EndPoint)
	if redisData := repo.redis.GetStringData(dump_server_env.GetRedisKey(id)); redisData != "" {
		if err := json.Unmarshal([]byte(redisData), result); err == nil {
			result.RestoreTimeStamp()
			return result, err
		}
	}
	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "id", Value: id,
	}}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("GeneralUser Entity identified by id %s not found", id),
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
		return nil, err
	}
	repo.cacheDataToRedis(result)
	return result, err
}

// func (repo *EndPointRepoStruct) FindOneUserNameAndBullionId(uname string, bullionId string) (*dump_server_interfaces.EndPoint, error) {
// 	var result dump_server_interfaces.EndPoint
// 	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
// 		Key: "userName", Value: uname,
// 	}, {
// 		Key: "bullionId", Value: bullionId,
// 	}}).Decode(&result)
// 	if err != nil {
// 		if errors.Is(err, mongo.ErrNoDocuments) {
// 			// This error means your query did not match any documents.
// 			err = &interfaces.RequestError{
// 				StatusCode: http.StatusBadRequest,
// 				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
// 				Message:    fmt.Sprintf("GeneralUser Entity identified by uname %s and bullionId %s not found", uname, bullionId),
// 				Name:       "ENTITY_NOT_FOUND",
// 			}
// 		} else {
// 			err = &interfaces.RequestError{
// 				StatusCode: 500,
// 				Code:       interfaces.ERROR_INTERNAL_SERVER,
// 				Message:    fmt.Sprintf("Internal Server Error: %s", err.Error()),
// 				Name:       "INTERNAL_ERROR",
// 			}
// 		}
// 		return nil, err
// 	}
// 	return &result, err
// }
