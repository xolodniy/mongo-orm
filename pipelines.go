package mongo_orm

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// legacy, just for example
func initPipeSortByPopular(likesTableName string, onlyActive bool, limit, offset int) mongo.Pipeline {
	pipe := mongo.Pipeline{
		{{"$addFields", bson.D{{"id", bson.D{{"$toString", "$_id"}}}}}},
		{{"$lookup", bson.D{
			{"from", likesTableName},
			{"localField", "id"},
			{"foreignField", "parent"},
			{"as", "likes"},
		}}},
		{{"$sort", bson.D{{"likes", -1}}}},
	}
	if onlyActive {
		pipe = append(pipe, bson.D{{"$match", bson.M{"active": true}}})
	}
	if offset > 0 {
		pipe = append(pipe, bson.D{{"$skip", offset}})
	}
	if limit > 0 {
		pipe = append(pipe, bson.D{{"$limit", limit}})
	}
	return pipe
}
