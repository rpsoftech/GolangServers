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
)

type GeneralUserReqRepoStruct struct {
	collection *mongo.Collection
}

const generalUserReqCollectionName = "GeneralUserReq"

var GeneralUserReqRepo *GeneralUserReqRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(generalUserReqCollectionName)
	GeneralUserReqRepo = &GeneralUserReqRepoStruct{
		collection: coll,
	}
	addComboUniqueIndexesToCollection([]string{"generalUserId", "bullionId"}, GeneralUserReqRepo.collection)
	addUniqueIndexesToCollection([]string{"id"}, GeneralUserReqRepo.collection)
}

func (repo *GeneralUserReqRepoStruct) Save(entity *bullion_main_server_interfaces.GeneralUserReqEntity) (*bullion_main_server_interfaces.GeneralUserReqEntity, error) {

	if err := utility_functions.ValidateStructAndReturnReqError(entity, &interfaces.RequestError{
		StatusCode: http.StatusBadRequest,
		Code:       interfaces.ERROR_INVALID_ENTITY,
		Message:    "",
		Name:       "ERROR_INVALID_ENTITY",
	}); err != nil {
		return entity, err
	}
	err := repo.collection.FindOneAndUpdate(mongodb.MongoCtx, bson.D{{
		Key: "_id", Value: entity.ID,
	}}, bson.D{{Key: "$set", Value: entity}}, findOneAndUpdateOptions).Err()
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

func (repo *GeneralUserReqRepoStruct) FindOneByGeneralUserIdAndBullionId(generalUserId string, bullionId string) (*bullion_main_server_interfaces.GeneralUserReqEntity, error) {
	var result bullion_main_server_interfaces.GeneralUserReqEntity
	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "generalUserId", Value: generalUserId,
	}, {
		Key: "bullionId", Value: bullionId,
	}}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("GeneralUserReq Entity identified by generalUserID %s and bullionId %s not found", generalUserId, bullionId),
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
func (repo *GeneralUserReqRepoStruct) FindOne(id string) (*bullion_main_server_interfaces.GeneralUserReqEntity, error) {
	var result bullion_main_server_interfaces.GeneralUserReqEntity

	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "id", Value: id,
	}}).Decode(&result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("GeneralUserReq Entity identified by id %s not found", id),
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
