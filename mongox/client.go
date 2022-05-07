package mongox

import (
	"context"
	"time"

	"github.com/visonlv/go-vkit/config"
	"github.com/visonlv/go-vkit/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoClient struct {
	c      *mongo.Client   // 客户端
	db     *mongo.Database // 数据库
	opTime time.Duration   // 每次操作(增删改查...)的最大时间
}

func NewDefault() (*MongoClient, error) {
	db := config.GetString("database.mongo.db")
	url := config.GetString("database.mongo.url")
	opMaxSecord := time.Duration(config.GetInt("database.mongo.op-max-secord"))
	if string

	return NewClient(url, db, opMaxSecord*time.Second)
}

func NewClient(uri, dbName string, withTimeout time.Duration) (*MongoClient, error) {
	opt := options.Client().ApplyURI(uri)
	// 创建客户端
	client, err := mongo.NewClient(opt)
	if err != nil {
		logger.Infof("[MongoClient] create client fail e:%s", err.Error())
		return nil, err
	}

	// 只是用来设置在多少秒必须完成 client.Connect 和 client.Ping 操作
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

	// 因为要一直保持 *mongo.Client 的存在好让它自己管理连接池
	// 所以不用 defer client.Disconnect(ctx)
	err = client.Connect(ctx)
	if err != nil {
		logger.Infof("[MongoClient] client connect  fail e:%s", err.Error())
		return nil, err
	}

	// ping 3秒 还不通就取消~
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		logger.Infof("[MongoClient] client ping fail e:%s", err.Error())
		return nil, err
	}

	mgoClient := &MongoClient{
		c:      client,
		db:     client.Database(dbName), // 应用数据库
		opTime: withTimeout,
	}

	return mgoClient, nil
}

func (the *MongoClient) GetClient() *mongo.Client {
	return the.c
}

func (the *MongoClient) GetDB() *mongo.Database {
	return the.db
}

func (the *MongoClient) Collection(colName string) *mongo.Collection {
	return the.db.Collection(colName)
}

func (the *MongoClient) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	return ctxWithTimeout, cancel
}

// 判断是否为 mongo.ErrNoDocuments 找不到文档
func (the *MongoClient) IsNoDocs(err error) bool {
	return err == mongo.ErrNoDocuments
}

// 如果是 mongo.ErrNoDocuments 返回 nil
func (the *MongoClient) ClearNoDocs(err error) error {
	if err == mongo.ErrNoDocuments {
		return nil
	}

	return err
}
