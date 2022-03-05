package mongo_orm

import (
	"fmt"
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

func errUnsupportedByMongoMapper(obj interface{}) error {
	return fmt.Errorf("object '%T' isn't supported. Please implement Collection() method for start using it", obj)
}
