package mongox

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 下面的封装是 mongodb api 进行封装，目的是添加超时取消设置~
func (the *MongoClient) InsertOne(ctx context.Context, collName string, doc interface{},
	opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).InsertOne(ctxWithTimeout, doc, opts...)
}

func (the *MongoClient) InsertMany(ctx context.Context, collName string, docs []interface{},
	opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).InsertMany(ctxWithTimeout, docs, opts...)
}

func (the *MongoClient) DeleteOne(ctx context.Context, collName string, filter interface{},
	opts ...*options.DeleteOptions) (int64, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	r, e := the.db.Collection(collName).DeleteOne(ctxWithTimeout, filter, opts...)
	return r.DeletedCount, e
}

func (the *MongoClient) DeleteMany(ctx context.Context, collName string, filter interface{},
	opts ...*options.DeleteOptions) (int64, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	r, e := the.db.Collection(collName).DeleteMany(ctxWithTimeout, filter, opts...)
	return r.DeletedCount, e
}

func (the *MongoClient) UpdateByID(ctx context.Context, collName string, id interface{}, update interface{},
	opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).UpdateByID(ctxWithTimeout, id, update, opts...)
}

func (the *MongoClient) UpdateOne(ctx context.Context, collName string, filter interface{}, update interface{},
	opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).UpdateOne(ctxWithTimeout, filter, update, opts...)
}

func (the *MongoClient) UpdateMany(ctx context.Context, collName string, filter interface{}, update interface{},
	opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).UpdateMany(ctxWithTimeout, filter, update, opts...)
}

func (the *MongoClient) ReplaceOne(ctx context.Context, collName string, filter interface{}, replacement interface{},
	opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).ReplaceOne(ctxWithTimeout, filter, replacement, opts...)
}

func (the *MongoClient) Aggregate(ctx context.Context, collName string, pipeline interface{},
	opts ...*options.AggregateOptions) (*mongo.Cursor, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel() // 这里取消了，但游标还是会存在的，所以游标要自己释放~

	return the.db.Collection(collName).Aggregate(ctxWithTimeout, pipeline, opts...)
}

func (the *MongoClient) CountDocuments(ctx context.Context, collName string, filter interface{},
	opts ...*options.CountOptions) (int64, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).CountDocuments(ctxWithTimeout, filter, opts...)
}

func (the *MongoClient) EstimatedDocumentCount(ctx context.Context, collName string,
	opts ...*options.EstimatedDocumentCountOptions) (int64, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).EstimatedDocumentCount(ctxWithTimeout, opts...)
}

func (the *MongoClient) Distinct(ctx context.Context, collName string, fieldName string, filter interface{},
	opts ...*options.DistinctOptions) ([]interface{}, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).Distinct(ctxWithTimeout, fieldName, fieldName, opts...)
}

func (the *MongoClient) Find(ctx context.Context, collName string, filter interface{},
	opts ...*options.FindOptions) (*mongo.Cursor, error) {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel() // 这里取消了，但游标还是会存在的，所以游标要自己释放~

	return the.db.Collection(collName).Find(ctxWithTimeout, filter, opts...)
}

func (the *MongoClient) FindOne(ctx context.Context, collName string, filter interface{},
	opts ...*options.FindOneOptions) *mongo.SingleResult {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).FindOne(ctxWithTimeout, filter, opts...)
}

func (the *MongoClient) FindOneAndDelete(ctx context.Context, collName string, filter interface{},
	opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).FindOneAndDelete(ctxWithTimeout, filter, opts...)
}

func (the *MongoClient) FindOneAndReplace(ctx context.Context, collName string, filter interface{},
	replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).FindOneAndReplace(ctxWithTimeout, filter, replacement, opts...)
}

func (the *MongoClient) FindOneAndUpdate(ctx context.Context, collName string, filter interface{},
	update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {

	ctxWithTimeout, cancel := context.WithTimeout(ctx, the.opTime)
	defer cancel()

	return the.db.Collection(collName).FindOneAndUpdate(ctxWithTimeout, filter, update, opts...)
}
