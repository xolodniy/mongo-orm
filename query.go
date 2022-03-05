// Raw bson builder
// TODO: elaborate intersect fields (similar fields currently replaces by different methods)
package mongo_orm

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Query struct {
	And []bson.D
}

func NewQuery() *Query {
	return &Query{}
}

func (q *Query) Exec() bson.M {
	if len(q.And) == 0 {
		return bson.M{}
	}
	return bson.M{"$and": q.And}
}

func QueryID(objectID primitive.ObjectID) bson.M {
	return bson.M{"_id": objectID}
}

// TextSearch find substring throw specified fields
func (q *Query) TextSearch(text string, fields ...string) {
	or := make([]bson.M, len(fields))
	for i := range fields {
		or[i] = bson.M{
			fields[i]: bson.M{
				"$regex":   text,
				"$options": "i",
			},
		}
	}
	q.And = append(q.And, bson.D{{"$or", or}})
}

func (q *Query) Equal(field string, value interface{}) {
	q.And = append(q.And, bson.D{{field, bson.M{"$eq": value}}})
}

func (q *Query) NotEqual(field string, value interface{}) {
	q.And = append(q.And, bson.D{{field, bson.M{"$ne": value}}})
}

func (q *Query) Empty(field string) {
	emptyConditions := []bson.D{
		{{field, bson.M{"$exists": false}}},
		{{field, nil}},
		{{field, bson.M{"$size": 0}}},
	}
	q.And = append(q.And, bson.D{{"$or", emptyConditions}})
}

func (q *Query) NotEmpty(field string) {
	q.And = append(q.And,
		bson.D{{
			field, bson.M{"$exists": true, "$ne": nil},
		}},
	)
	//q.And = append(q.And, bson.D{{field, bson.M{"$size": bson.M{"$gt": 0}}}})
}

func (q *Query) Add(field string, value interface{}) {
	q.And = append(q.And, bson.D{{field, value}})
}

func (q *Query) IDs(ids []string) {
	var bsonIDs = make([]primitive.ObjectID, 0, len(ids))
	for i := range ids {
		if bsonID, err := primitive.ObjectIDFromHex(ids[i]); err == nil {
			bsonIDs = append(bsonIDs, bsonID)
		}
	}
	q.And = append(q.And, bson.D{{
		"_id", bson.D{{"$in", bsonIDs}},
	}})
}

func (q *Query) FieldAnyOf(field string, anyOf ...string) {
	q.And = append(q.And, bson.D{{
		field, bson.D{{"$in", anyOf}},
	}})
}
