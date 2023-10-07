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

func (the *MysqlClient) Where(query any, args ...any) *MysqlClient {
	tx := the.db.Where(query, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Model(value any) *MysqlClient {
	tx := the.db.Model(value)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Count(count *int64) *MysqlClient {
	tx := the.db.Count(count)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Table(name string, args ...any) *MysqlClient {
	tx := the.db.Table(name, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Distinct(args ...any) *MysqlClient {
	tx := the.db.Distinct(args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Select(query any, args ...any) *MysqlClient {
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

func (the *MysqlClient) Find(dest any, conds ...any) *MysqlClient {
	tx := the.db.Find(dest, conds...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) First(dest any, conds ...any) *MysqlClient {
	tx := the.db.First(dest, conds...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Raw(sql string, values ...any) *MysqlClient {
	tx := the.db.Raw(sql, values...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Rows() (*sql.Rows, error) {
	return the.db.Rows()
}

func (the *MysqlClient) ScanRows(rows *sql.Rows, dest any) error {
	return the.db.ScanRows(rows, dest)
}

func (the *MysqlClient) Joins(query string, args ...any) *MysqlClient {
	tx := the.db.Joins(query, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) AutoMigrate(dst ...any) error {
	return the.db.AutoMigrate(dst...)
}

func (the *MysqlClient) Delete(value any, conds ...any) *MysqlClient {
	tx := the.db.Delete(value, conds...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Update(column string, value any) *MysqlClient {
	tx := the.db.Update(column, value)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Order(value any) *MysqlClient {
	tx := the.db.Order(value)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Having(query any, args ...any) *MysqlClient {
	tx := the.db.Having(query, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Or(query any, args ...any) *MysqlClient {
	tx := the.db.Or(query, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Not(query any, args ...any) *MysqlClient {
	tx := the.db.Not(query, args...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Omit(columns ...string) *MysqlClient {
	tx := the.db.Omit(columns...)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Scan(dest any) *MysqlClient {
	tx := the.db.Scan(dest)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Pluck(column string, dest any) *MysqlClient {
	tx := the.db.Pluck(column, dest)
	return newOnChain(the, tx)
}

func (the *MysqlClient) Take(dest any, conds ...any) *MysqlClient {
	tx := the.db.Take(dest, conds...)
	return newOnChain(the, tx)
}
