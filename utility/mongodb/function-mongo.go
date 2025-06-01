package mongodb

import (
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDbFilter struct {
	Conditions *bson.D
	Sort       *bson.D
	Limit      int64
	Skip       int64
}

var FindOneAndUpdateOptions = &options.FindOneAndUpdateOptions{
	Upsert: utility_functions.BoolPointer(true),
}

func AddComboUniqueIndexesToCollection(UniqueIndexes []string, collection *mongo.Collection) {
	i := bson.D{}
	for _, element := range UniqueIndexes {
		i = append(i, bson.E{Key: element, Value: 1})
	}
	collection.Indexes().CreateOne(MongoCtx, mongo.IndexModel{
		Keys:    i,
		Options: options.Index().SetUnique(true),
	})
}

func AddIndexesToCollection(indexesString []string, collection *mongo.Collection) {
	indexArray := make([]mongo.IndexModel, len(indexesString))
	for i, element := range indexesString {
		indexArray[i] = mongo.IndexModel{
			Keys:    bson.D{{Key: element, Value: 1}},
			Options: options.Index().SetUnique(false),
		}
	}
	collection.Indexes().CreateMany(MongoCtx, indexArray)
}

func AddUniqueIndexesToCollection(UniqueIndexes []string, collection *mongo.Collection) {
	indexes := make([]mongo.IndexModel, len(UniqueIndexes))
	for i, element := range UniqueIndexes {
		indexes[i] = mongo.IndexModel{
			Keys:    bson.D{{Key: element, Value: 1}},
			Options: options.Index().SetUnique(true),
		}
	}
	collection.Indexes().CreateMany(MongoCtx, indexes)
}

func AddComboIndexesToCollection(UniqueIndexes []string, collection *mongo.Collection) {
	i := bson.D{}
	for _, element := range UniqueIndexes {
		i = append(i, bson.E{Key: element, Value: 1})
	}
	collection.Indexes().CreateOne(MongoCtx, mongo.IndexModel{
		Keys: i,
	})
}
