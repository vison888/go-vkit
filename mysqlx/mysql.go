package mysqlx

import (
	"database/sql"

	"gorm.io/gorm"
)

func (the *MysqlClient) Transaction(fc func(tx *MysqlClient) error, opts ...*sql.TxOptions) error {
	return the.db.Transaction(func(tx *gorm.DB) error {
		return fc(NewOnChain(the, tx))
	}, opts...)
}

func (the *MysqlClient) Where(query interface{}, args ...interface{}) *MysqlClient {
	tx := the.db.Where(query, args...)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Model(value interface{}) *MysqlClient {
	tx := the.db.Model(value)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Count(count *int64) *MysqlClient {
	tx := the.db.Count(count)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Table(name string, args ...interface{}) *MysqlClient {
	tx := the.db.Table(name, args...)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Distinct(args ...interface{}) *MysqlClient {
	tx := the.db.Distinct(args...)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Select(query interface{}, args ...interface{}) *MysqlClient {
	tx := the.db.Select(query, args...)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Group(name string) *MysqlClient {
	tx := the.db.Group(name)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Offset(offset int) *MysqlClient {
	tx := the.db.Offset(offset)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Limit(limit int) *MysqlClient {
	tx := the.db.Limit(limit)
	return NewOnChain(the, tx)
}

func (the *MysqlClient) Find(dest interface{}, conds ...interface{}) *MysqlClient {
	tx := the.db.Find(dest, conds...)
	return NewOnChain(the, tx)
}
