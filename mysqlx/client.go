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

func NewDefault() (*MysqlClient, error) {
	driverClassName := config.GetString("database.mysql.driverClassName")
	username := config.GetString("database.mysql.username")
	password := config.GetString("database.mysql.password")
	url := config.GetString("database.mysql.url")
	return NewClient(driverClassName, username, password, url)
}

func NewClient(driverClassName, username, password, url string) (*MysqlClient, error) {
	if driverClassName == "" || username == "" || password == "" || url == "" {
		logger.Errorf("[mysqlx] driverClassName:%s username:%s password:%s url:%s has empty", driverClassName, username, password, url)
		return nil, errors.New("param error")
	}

	url = fmt.Sprintf("%s:%s@%s", username, password, url)
	logger.Info("[mysqlx] mysql url=:" + url)
	db, err := gorm.Open(mysql.New(mysql.Config{
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
		DriverName:                driverClassName,
		DSN:                       url,
	}), &gorm.Config{})

	if err != nil {
		logger.Errorf("[mysqlx] open fail err:%s", err.Error())
		return nil, errors.New("open fail")
	}
	// 获取通用数据库对象 sql.DB ，然后使用其提供的功能
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf("[mysqlx] db.DB fail err:%s", err.Error())
		return nil, errors.New("db.DB fail")
	}

	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	lifeTime := 3600
	sqlDB.SetConnMaxLifetime(time.Duration(lifeTime) * time.Second)

	logger.Errorf("[mysqlx] driverClassName:%s username:%s password:%s url:%s start success", driverClassName, username, password, url)

	c := &MysqlClient{db: db}
	return c, nil
}

func (c *MysqlClient) GetDB() *gorm.DB {
	return c.db
}
