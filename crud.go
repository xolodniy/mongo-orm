package mongo_orm

import (
	"errors"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xolodniy/pretty"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CRUD interface {
	GetMany(result interface{}, searchText *string, offset, limit int) error
	GetManyByIDs(result interface{}, ids []string) error
	GetByID(result interface{}, id string) error
	Create(i interface{}) error
	Update(i interface{}, id string) error
	Delete(i interface{}, id string) error
	Count(i interface{}, searchText *string) (int, error)

	// internal package usage only
	// TODO: Make it public (think about encapsulate FindOptions && prepare use Query object outside)
	getMany(result interface{}, query *Query, opts *options.FindOptions) error
	get(result interface{}, query *Query) error
	delete(i interface{}, query *Query) error
}

func (m *Model) GetMany(result interface{}, searchText *string, offset, limit int) error {
	query := NewQuery()
	if searchText != nil {
		// TODO: init field customization for search
		query.TextSearch(*searchText, "title")
	}
	opts := options.Find().SetSkip(int64(offset)).SetLimit(int64(limit))
	return m.getMany(result, query, opts)
}

//
func (m *Model) Create(i interface{}) error {
	t, ok := i.(mongoMapper)
	if !ok {
		return errUnsupportedByMongoMapper(i)
	}
	obj := reflect.ValueOf(i).Elem()
	if obj.Kind() != reflect.Struct {
		logrus.WithField("object", pretty.Print(i)).Error("can't create object, only structs supported")
		return m.errInternal
	}

	// init default fields
	f := obj.FieldByName("ID")
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.Array {
		f.Set(reflect.ValueOf(primitive.NewObjectID()))
	}
	f = obj.FieldByName("CreatedAt")
	if f.IsValid() && f.CanSet() {
		f.Set(reflect.ValueOf(time.Now()))
	}
	f = obj.FieldByName("UpdatedAt")
	if f.IsValid() && f.CanSet() {
		f.Set(reflect.ValueOf(time.Now()))
	}
	_, err := m.db.Collection(t.Collection()).InsertOne(m.ctx, i)
	if err != nil {
		logrus.WithError(err).WithField("object", pretty.Print(i)).Error("can't create object")
		return m.errInternal
	}
	return nil
}

func (m *Model) GetByID(result interface{}, id string) error {
	bsonID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errInvalidID
	}
	query := NewQuery()
	query.Add("_id", bsonID)
	return m.get(result, query)
}

func (m *Model) Update(i interface{}, id string) error {
	t, ok := i.(mongoMapper)
	if !ok {
		return errUnsupportedByMongoMapper(i)
	}
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errInvalidID
	}
	obj := reflect.ValueOf(i).Elem()
	if obj.Kind() != reflect.Struct {
		logrus.WithField("object", pretty.Print(i)).Error("can't update object, only structs supported")
		return m.errInternal
	}
	f := obj.FieldByName("UpdatedAt")
	if f.IsValid() && f.CanSet() {
		f.Set(reflect.ValueOf(time.Now()))
	}

	_, err = m.db.Collection(t.Collection()).ReplaceOne(m.ctx, QueryID(objectID), i)
	if err != nil {
		logrus.WithError(err).
			WithField("id", id).
			WithField("object", pretty.Print(i)).
			Error("can't update object")
		return m.errInternal
	}
	return nil
}

func (m *Model) Delete(i interface{}, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errInvalidID
	}
	query := NewQuery()
	query.Add("_id", objectID)
	return m.delete(i, query)
}

func (m *Model) Count(i interface{}, searchText *string) (int, error) {
	var query = NewQuery()
	if searchText != nil {
		// TODO: init field customization
		query.TextSearch(*searchText, "title")
	}
	return m.count(i, query)
}

func (m *Model) GetManyByIDs(result interface{}, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := NewQuery()
	query.IDs(ids)
	return m.getMany(result, query, nil)
}

func (m *Model) count(i interface{}, query *Query) (int, error) {
	t, ok := i.(mongoMapper)
	if !ok {
		return 0, errUnsupportedByMongoMapper(i)
	}
	count, err := m.db.Collection(t.Collection()).CountDocuments(m.ctx, query.Exec())
	if err != nil {
		logrus.WithError(err).Errorf("can't count objects")
		return 0, m.errInternal
	}
	return int(count), nil
}

func (m *Model) getMany(result interface{}, query *Query, opts *options.FindOptions) error {
	obj := reflect.TypeOf(result).Elem()
	if obj.Kind() != reflect.Slice {
		return errors.New("result object should be slice")
	}
	t, ok := reflect.New(obj.Elem()).Interface().(mongoMapper)
	if !ok {
		return errUnsupportedByMongoMapper(result)
	}

	cursor, err := m.db.Collection(t.Collection()).Find(m.ctx, query.Exec(), opts)
	if err != nil {
		logrus.WithError(err).Error("can't query objects")
		return m.errInternal
	}
	err = cursor.All(m.ctx, result)
	if err != nil {
		logrus.WithError(err).Error("can't decode queried objects")
		return m.errInternal
	}
	return nil
}

func (m *Model) get(result interface{}, query *Query) error {
	t, ok := result.(mongoMapper)
	if !ok {
		return errUnsupportedByMongoMapper(result)
	}
	err := m.db.Collection(t.Collection()).FindOne(m.ctx, query.Exec()).Decode(result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return m.errNotFound
	}
	if err != nil {
		logrus.WithError(err).Error("can't get object from database")
		return m.errInternal
	}
	return nil
}

func (m *Model) delete(i interface{}, query *Query) error {
	t, ok := i.(mongoMapper)
	if !ok {
		return errUnsupportedByMongoMapper(i)
	}
	_, err := m.db.Collection(t.Collection()).DeleteOne(m.ctx, query.Exec())
	if err != nil {
		logrus.WithError(err).Error("can't delete object from database")
		return m.errInternal
	}
	return nil
}

func (m *Model) aggregate(result interface{}, pipe mongo.Pipeline, opts *options.AggregateOptions) error {
	obj := reflect.TypeOf(result).Elem()
	if obj.Kind() != reflect.Slice {
		return errors.New("result object should be slice")
	}
	t, ok := reflect.New(obj.Elem()).Interface().(mongoMapper)
	if !ok {
		return errUnsupportedByMongoMapper(result)
	}

	cursor, err := m.db.Collection(t.Collection()).Aggregate(m.ctx, pipe, opts)
	if err != nil {
		logrus.WithError(err).Error("can't get objects")
		return m.errInternal
	}
	err = cursor.All(m.ctx, result)
	if err != nil {
		logrus.WithError(err).Error("can't decode objects")
		return m.errInternal
	}
	return nil
}
