package mysqlx

import (
	"errors"
	"fmt"
	"time"

	"github.com/visonlv/go-vkit/config"
	"github.com/visonlv/go-vkit/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlClient struct {
	db    *gorm.DB // 客户端
	clone bool     // 是否clone
}

func NewOnChain(client *MysqlClient, db *gorm.DB) *MysqlClient {
	if client.clone {
		return client
	}
	return &MysqlClient{db: db, clone: true}
}

func NewDefault() *MysqlClient {
	database := config.GetString("database.mysql.database")
	username := config.GetString("database.mysql.username")
	password := config.GetString("database.mysql.password")
	url := config.GetString("database.mysql.url")
	maxConn := config.GetInt("database.mysql.maxConn", 100)
	maxIdel := config.GetInt("database.mysql.maxIdel", 10)
	maxLifeTime := config.GetInt("database.mysql.maxLifeTime", 3600)
	return NewClient(database, username, password, url, maxConn, maxIdel, maxLifeTime)
}

func NewClient(database, username, password, url string, maxConn, maxIdel, maxLifeTime int) *MysqlClient {
	if database == "" || username == "" || password == "" || url == "" {
		logger.Errorf("[mysqlx] database:%s username:%s password:%s url:%s has empty", database, username, password, url)
		panic(errors.New("param error"))
	}

	url = fmt.Sprintf(url, database)
	url = fmt.Sprintf("%s:%s@%s", username, password, url)
	db, err := gorm.Open(mysql.New(mysql.Config{
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
		DriverName:                "mysql",
		DSN:                       url,
	}), &gorm.Config{})

	if err != nil {
		logger.Errorf("[mysqlx] open fail err:%s", err.Error())
		panic(err)
	}
	// 获取通用数据库对象 sql.DB ，然后使用其提供的功能
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf("[mysqlx] db.DB fail err:%s", err.Error())
		panic(err)
	}

	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(maxIdel)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(maxConn)
	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	lifeTime := maxLifeTime
	sqlDB.SetConnMaxLifetime(time.Duration(lifeTime) * time.Second)

	logger.Infof("[mysqlx] database:%s username:%s password:%s url:%s start success", database, username, password, url)

	c := &MysqlClient{db: db}
	return c
}

func (c *MysqlClient) GetDB() *gorm.DB {
	return c.db
}
