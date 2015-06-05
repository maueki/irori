package db

import (
	"github.com/coopernurse/gorp"
	"runtime"
)

type DbTransaction struct {
	Transaction *gorp.Transaction
	Error       error
}

func CreateTransaction(DbMap *gorp.DbMap) (*DbTransaction, error) {
	t := DbTransaction{}
	trans, err := DbMap.Begin()

	if err != nil {
		return nil, err
	}

	t.Transaction = trans
	runtime.SetFinalizer(&t, finalizer)

	return &t, nil
}

func finalizer(t *DbTransaction) {
	t.Transaction.Rollback()
}

func (t *DbTransaction) Insert(list ...interface{}) *DbTransaction {
	if t.Error == nil {
		t.Error = t.Transaction.Insert(list...)
	}

	return t
}

func (t *DbTransaction) Update(list ...interface{}) *DbTransaction {
	if t.Error == nil {
		_, t.Error = t.Transaction.Update(list...)
	}

	return t
}

func (t *DbTransaction) Delete(list ...interface{}) *DbTransaction {
	if t.Error == nil {
		_, t.Error = t.Transaction.Delete(list...)
	}

	return t
}

func (t *DbTransaction) Subscribe() error {
	if t.Error != nil {
		t.Transaction.Rollback()
		return t.Error
	}

	return t.Transaction.Commit()
}
