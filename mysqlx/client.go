package mysqlx

import (
	"errors"
	"time"

	"github.com/visonlv/go-vkit/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlClient struct {
	db    *gorm.DB // 客户端
	clone bool     // 是否clone
}

func newOnChain(client *MysqlClient, db *gorm.DB) *MysqlClient {
	if client.clone {
		return client
	}
	return &MysqlClient{db: db, clone: true}
}

func NewClient(uri string, maxConn, maxIdel, maxLifeTime int) (*MysqlClient, error) {
	if uri == "" || maxConn == 0 || maxIdel == 0 || maxLifeTime == 0 {
		logger.Errorf("[mysqlx] NewClient fail:param error uri:%s maxConn:%d maxIdel:%d maxLifeTime:%d", uri, maxConn, maxIdel, maxLifeTime)
		return nil, errors.New("param error")
	}

	db, err := gorm.Open(mysql.New(mysql.Config{
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
		DriverName:                "mysql",
		DSN:                       uri,
	}), &gorm.Config{})

	if err != nil {
		logger.Errorf("[mysqlx] NewClient fail:%s uri:%s maxConn:%d maxIdel:%d maxLifeTime:%d", err.Error(), uri, maxConn, maxIdel, maxLifeTime)
		return nil, err
	}
	// 获取通用数据库对象 sql.DB ，然后使用其提供的功能
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf("[mysqlx] NewClient fail:%s uri:%s maxConn:%d maxIdel:%d maxLifeTime:%d", err.Error(), uri, maxConn, maxIdel, maxLifeTime)
		return nil, err
	}

	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(maxIdel)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(maxConn)
	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	lifeTime := maxLifeTime
	sqlDB.SetConnMaxLifetime(time.Duration(lifeTime) * time.Second)

	c := &MysqlClient{db: db}

	logger.Infof("[mysqlx] NewClient success uri:%s maxConn:%d maxIdel:%d maxLifeTime:%d", uri, maxConn, maxIdel, maxLifeTime)
	return c, nil
}

func (c *MysqlClient) GetDB() *gorm.DB {
	return c.db
}
