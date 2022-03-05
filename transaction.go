package mongo_orm

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

type TX interface {
	StartTransaction() (*Model, error)
	Commit() error
	Rollback(err error) error
}

func (m *Model) StartTransaction() (*Model, error) {
	session, err := m.db.Client().StartSession()
	if err != nil {
		logrus.WithError(err).Error("can't init session for transaction")
		return nil, m.errInternal
	}
	tx := &Model{
		db:          m.db,
		errInternal: m.errInternal,
		errNotFound: m.errNotFound,
		ctx:         mongo.NewSessionContext(context.Background(), session),
	}
	err = tx.ctx.StartTransaction()
	if err != nil {
		logrus.WithError(err).Error("can't start transaction")
		return nil, m.errInternal
	}
	return tx, nil
}

func (tx *Model) Commit() error {
	if err := tx.ctx.CommitTransaction(tx.ctx); err != nil {
		logrus.WithError(err).Error("can't commit transaction")
		return tx.errInternal
	}
	return nil
}

func (tx *Model) Rollback(err error) error {
	if err := tx.ctx.AbortTransaction(tx.ctx); err != nil {
		logrus.WithError(err).Error("error on rollback transaction")
	}
	return tx.errInternal
}
