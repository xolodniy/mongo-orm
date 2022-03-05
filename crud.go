package mongo_orm

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
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
	// TODO: Make it public (think solution encapsulate FindOptions)
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
		logrus.WithField("objectType", fmt.Sprintf("%T", i))
		return errors.New("unsupported result object")
	}
	obj := reflect.ValueOf(i).Elem()
	if obj.Kind() != reflect.Struct {
		logrus.
			WithField("iValue", fmt.Sprintf("%+v", i)).
			WithField("iType", fmt.Sprintf("%T", i)).
			Error("can't create value in database")
		return m.errInternal
	}
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
		logrus.WithError(err).Error("can't create object in database")
		return m.errInternal
	}
	return nil
}

func (m *Model) GetByID(result interface{}, id string) error {
	bsonID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid object id")
	}
	query := NewQuery()
	query.Add("_id", bsonID)
	return m.get(result, query)
}

func (m *Model) Update(i interface{}, id string) error {
	t, ok := i.(mongoMapper)
	if !ok {
		logrus.WithField("objectType", fmt.Sprintf("%T", i))
		return errors.New("unsupported update object type")
	}
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid object id")
	}
	obj := reflect.ValueOf(i).Elem()
	if obj.Kind() != reflect.Struct {
		logrus.
			WithField("iValue", fmt.Sprintf("%+v", i)).
			WithField("iType", fmt.Sprintf("%T", i)).
			Error("can't update value in database")
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
			WithField("objectValue", fmt.Sprintf("%+v", i)).
			WithField("objectType", fmt.Sprintf("%T", i)).
			Error("can't update object by id")
		return m.errInternal
	}
	return nil
}

func (m *Model) Delete(i interface{}, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid tool id")
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
		logrus.WithField("objectType", fmt.Sprintf("%T", i)).Error("can't count objects")
		return 0, errors.New("unsupported count object type")
	}
	count, err := m.db.Collection(t.Collection()).CountDocuments(m.ctx, query.Exec())
	if err != nil {
		logrus.WithError(err).Errorf("can't count objects in database")
		return 0, m.errInternal
	}
	return int(count), nil
}

func (m *Model) getMany(result interface{}, query *Query, opts *options.FindOptions) error {
	obj := reflect.TypeOf(result).Elem()
	if obj.Kind() != reflect.Slice {
		logrus.WithField("objectType", fmt.Sprintf("%T", result))
		return errors.New("result object should be slice")
	}
	t, ok := reflect.New(obj.Elem()).Interface().(mongoMapper)
	if !ok {
		return errors.New("unsupported object type")
	}

	cursor, err := m.db.Collection(t.Collection()).Find(m.ctx, query.Exec(), opts)
	if err != nil {
		logrus.WithError(err).Error("can't Query objects")
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
		logrus.WithField("objectType", fmt.Sprintf("%T", result)).Error("can't get object by id")
		return errors.New("unsupported result object")
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
		logrus.WithField("objectType", fmt.Sprintf("%T", i)).Error("cant delete object from database")
		return errors.New("unsupported delete object type")
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
		logrus.WithField("objectType", fmt.Sprintf("%T", result))
		return errors.New("result object should be slice")
	}
	t, ok := reflect.New(obj.Elem()).Interface().(mongoMapper)
	if !ok {
		return errors.New("unsupported object type")
	}

	cursor, err := m.db.Collection(t.Collection()).Aggregate(m.ctx, pipe, opts)
	if err != nil {
		logrus.WithError(err).Error("can't get tools")
		return m.errInternal
	}
	err = cursor.All(m.ctx, result)
	if err != nil {
		logrus.WithError(err).Error("can't decode tools")
		return m.errInternal
	}
	return nil
}
