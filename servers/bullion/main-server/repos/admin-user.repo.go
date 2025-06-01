package bullion_main_server_repos

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AdminUserRepoStruct struct {
	collection *mongo.Collection
}

const adminUserCollectionName = "AdminUser"

var AdminUserRepo *AdminUserRepoStruct

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	coll := mongodb.MongoDatabase.Collection(adminUserCollectionName)
	AdminUserRepo = &AdminUserRepoStruct{
		collection: coll,
	}
	mongodb.AddUniqueIndexesToCollection([]string{"id"}, AdminUserRepo.collection)
	mongodb.AddComboUniqueIndexesToCollection([]string{"userName", "bullionId"}, AdminUserRepo.collection)
}

func (repo *AdminUserRepoStruct) Save(entity *bullion_main_server_interfaces.AdminUserEntity) (*bullion_main_server_interfaces.AdminUserEntity, error) {
	var result bullion_main_server_interfaces.AdminUserEntity
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
	return &result, err
}

func (repo *AdminUserRepoStruct) FindOne(id string) (*bullion_main_server_interfaces.AdminUserEntity, error) {
	var result bullion_main_server_interfaces.AdminUserEntity
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
	}
	return &result, err
}

func (repo *AdminUserRepoStruct) FindOneUserNameAndBullionId(uname string, bullionId string) (*bullion_main_server_interfaces.AdminUserEntity, error) {
	var result bullion_main_server_interfaces.AdminUserEntity
	err := repo.collection.FindOne(mongodb.MongoCtx, bson.D{{
		Key: "userName", Value: uname,
	}, {
		Key: "bullionId", Value: bullionId,
	}}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// This error means your query did not match any documents.
			err = &interfaces.RequestError{
				StatusCode: http.StatusBadRequest,
				Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
				Message:    fmt.Sprintf("GeneralUser Entity identified by uname %s and bullionId %s not found", uname, bullionId),
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
