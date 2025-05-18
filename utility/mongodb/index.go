package mongodb

import (
	"context"

	"github.com/rpsoftech/golang-servers/env"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBConfig struct {
	DB_URL  string `json:"DB_URL" validate:"required,url"`
	DB_NAME string `json:"DB_NAME_KEY" validate:"required,min=3"`
}

var MongoClient *mongo.Client
var MongoDatabase *mongo.Database
var MongoCtx = context.TODO()

func init() {
	if env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		return
	}
	config := &MongoDBConfig{
		DB_URL:  env.Env.GetEnv(env.MONGO_URL_KEY),
		DB_NAME: env.Env.GetEnv(env.MONGO_DB_NAME_KEY),
	}
	env.ValidateEnv(config)
	// env.Env.DB_URL
	client, err := mongo.Connect(MongoCtx, options.Client().ApplyURI(config.DB_URL).SetMinPoolSize(2))
	if err != nil {
		panic(err)
	}
	MongoClient = client
	MongoDatabase = client.Database(config.DB_NAME)
	go func() {
		err := client.Ping(MongoCtx, nil)
		if err != nil {
			panic(err)
		}
	}()
}

func DeferFunction() {
	if err := MongoClient.Disconnect(context.TODO()); err != nil {
		panic(err)
	}
}
