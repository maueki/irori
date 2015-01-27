package db

import (
	"github.com/coopernurse/gorp"
	"runtime"
)

type DbAccessor struct {
	Transaction *gorp.Transaction
	Error       error
}

func CreateAccessor(DbMap *gorp.DbMap) (*DbAccessor, error) {
	t := DbAccessor{}
	trans, err := DbMap.Begin()

	if err != nil {
		return nil, err
	}

	t.Transaction = trans
	runtime.SetFinalizer(&t, finalizer)

	return &t, nil
}

func finalizer(t *DbAccessor) {
	t.Transaction.Rollback()
}

func (t *DbAccessor) Insert(list ...interface{}) *DbAccessor {
	if t.Error == nil {
		t.Error = t.Transaction.Insert(list...)
	}

	return t
}

func (t *DbAccessor) Update(list ...interface{}) *DbAccessor {
	if t.Error == nil {
		_, t.Error = t.Transaction.Update(list...)
	}

	return t
}

func (t *DbAccessor) Delete(list ...interface{}) *DbAccessor {
	if t.Error == nil {
		_, t.Error = t.Transaction.Delete(list...)
	}

	return t
}

func (t *DbAccessor) Subscribe() error {
	if t.Error != nil {
		t.Transaction.Rollback()
		return t.Error
	}

	return t.Transaction.Commit()
}
