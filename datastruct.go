package mongo_orm

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// interface specify object-related collection name
type mongoMapper interface {
	Collection() string
}

type DefaultClaims struct {
	ID        primitive.ObjectID `bson:"_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

type ExampleObject struct {
	DefaultClaims `bson:",inline"`

	Title string `bson:"title"`
}

func (ExampleObject) Collection() string {
	return "example_objects"
}
