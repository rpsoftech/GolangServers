package bullion_main_server_repos

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"github.com/rpsoftech/golang-servers/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type BankRateCalcRepoStruct struct {
	collection *mongo.Collection
	redis      *redis.RedisClientStruct
	// historyCollection *mongo.Collection
}

const bankRateCalcRepoCollectionName = "BankRateCalc"
const bankRateRedisCollection = "bankRate"

// const bankRateCalcHistoryRepoCollectionName = "BankRateCalcHistory"

var BankRateCalcRepo *BankRateCalcRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(bankRateCalcRepoCollectionName)
	BankRateCalcRepo = &BankRateCalcRepoStruct{
		collection: coll,
		redis:      redis.InitRedisAndRedisClient(),
	}
	mongodb.AddUniqueIndexesToCollection([]string{"id", "bullionId"}, BankRateCalcRepo.collection)
}

func (repo *BankRateCalcRepoStruct) cacheDataToRedis(entity *bullion_main_server_interfaces.BankRateCalcEntity) {
	go redis.CacheDataToRedis(repo.redis, &entity, fmt.Sprintf("%s/%s", bankRateRedisCollection, entity.BullionId), redis.TimeToLive_OneDay)
}

func (repo *BankRateCalcRepoStruct) Save(entity *bullion_main_server_interfaces.BankRateCalcEntity) (*bullion_main_server_interfaces.BankRateCalcEntity, error) {
	var result bullion_main_server_interfaces.BankRateCalcEntity
	err := repo.collection.FindOneAndUpdate(mongodb.MongoCtx, bson.D{{
		Key: "_id", Value: entity.ID,
	}}, bson.D{{Key: "$set", Value: entity}}, mongodb.FindOneAndUpdateOptions).Decode(&result)
	entity.Updated()
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

func (repo *BankRateCalcRepoStruct) FindOneByBullionId(id string) (*bullion_main_server_interfaces.BankRateCalcEntity, error) {
	result := new(bullion_main_server_interfaces.BankRateCalcEntity)
	if redisData := repo.redis.GetStringData(fmt.Sprintf("%s/%s", bankRateRedisCollection, id)); redisData != "" {
		if err := json.Unmarshal([]byte(redisData), result); err == nil {
			result.RestoreTimeStamp()
			return result, err
		}
	}
	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "bullionId", Value: id,
	}}).Decode(result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("Bullion Entity identified by id %s not found", id),
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
