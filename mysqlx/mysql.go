package mysqlx

import (
	"database/sql"

	"gorm.io/gorm"
)

func (the *MysqlClient) Transaction(fc func(tx *MysqlClient) error, opts ...*sql.TxOptions) error {
	return the.db.Transaction(func(tx *gorm.DB) error {
		return fc(newOnChain(the, tx))
	}, opts...)
}

func (the *MysqlClient) Where(query interface{}, args ...interface{}) *MysqlClient {
	tx := the.db.Where(query, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Model(value interface{}) *MysqlClient {
	tx := the.db.Model(value)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Count(count *int64) *MysqlClient {
	tx := the.db.Count(count)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Table(name string, args ...interface{}) *MysqlClient {
	tx := the.db.Table(name, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Distinct(args ...interface{}) *MysqlClient {
	tx := the.db.Distinct(args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Select(query interface{}, args ...interface{}) *MysqlClient {
	tx := the.db.Select(query, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Group(name string) *MysqlClient {
	tx := the.db.Group(name)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Offset(offset int) *MysqlClient {
	tx := the.db.Offset(offset)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Limit(limit int) *MysqlClient {
	tx := the.db.Limit(limit)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Find(dest interface{}, conds ...interface{}) *MysqlClient {
	tx := the.db.Find(dest, conds...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) First(dest interface{}, conds ...interface{}) *MysqlClient {
	tx := the.db.First(dest, conds...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Raw(sql string, values ...interface{}) *MysqlClient {
	tx := the.db.Raw(sql, values...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Rows() (*sql.Rows, error) {
	return the.db.Rows()
}

func (the *MysqlClient) ScanRows(rows *sql.Rows, dest interface{}) error {
	return the.db.ScanRows(rows, dest)
}

func (the *MysqlClient) Joins(query string, args ...interface{}) *MysqlClient {
	tx := the.db.Joins(query, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) AutoMigrate(dst ...interface{}) error {
	return the.db.AutoMigrate(dst...)
}
