package mongo_orm

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	errInvalidID = errors.New("invalid identifier")

	fErrInvalidID = func(id string) error { return fmt.Errorf("invalid identifier '%s'", id) }
)

type Model struct {
	db *mongo.Database

	errInternal, errNotFound error

	// ctx used to make transactions
	ctx mongo.SessionContext
}

type Config struct {
	Name     string
	Host     string
	Port     int
	User     string
	Password string
}

func New(
	config Config,
	exampleErrNotFound error,
	exampleErrInternal error,
) *Model {
	// replica set required for use transactions
	url := fmt.Sprintf("mongodb://%s:%d/?replicaSet=rs0",
		config.Host,
		config.Port,
	)
	credential := options.Credential{
		Username: config.User,
		Password: config.Password,
	}
	opts := options.Client().ApplyURI(url).SetAuth(credential)
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		logrus.WithError(err).WithField("connURL", url).Fatal("Failed connect to database")
		return nil
	}
	db := client.Database(config.Name)
	m := &Model{
		db:          db,
		errInternal: exampleErrInternal,
		errNotFound: exampleErrNotFound,
	}
	return m
}
