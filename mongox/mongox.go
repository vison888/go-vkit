package mongox

import (
	"context"
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (the *MongoClient) DeleteById(ctx context.Context, collName string, id ...any) (int64, error) {
	ids := toSlice(id...)
	if len(ids) == 1 {
		filter := bson.M{"_id": ids[0]}
		return the.DeleteOne(ctx, collName, filter)
	} else if len(ids) == 0 {
		return 0, nil
	} else {
		filter := bson.M{"_id": bson.M{"$in": ids}}
		return the.DeleteMany(ctx, collName, filter)
	}
}

func (the *MongoClient) QueryById(ctx context.Context, collName string, id, pResult any) error {

	filter := bson.M{"_id": id}

	r := the.FindOne(ctx, collName, filter)
	err := r.Err()
	if err != nil {
		return err
	}
	return r.Decode(pResult)
}

func (the *MongoClient) DistinctQuery(ctx context.Context, colName, distinctField string, filter, result any) error {
	array, err := the.Distinct(ctx, colName, distinctField, filter)
	if err != nil {
		return err
	}
	//TODO 优化点：需要一个更好的处理方法
	b, err := json.Marshal(array)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, result)
	return err
}

func (the *MongoClient) QueryLatest(ctx context.Context, colName, sortBy string, filter, pResult any) error {
	return the.FindOne(ctx, colName, filter,
		options.FindOne().SetSort(sort([]string{sortBy}))).Decode(pResult)
}

func (the *MongoClient) QueryOne(ctx context.Context, colName string, filter, pResult any) error {
	return the.FindOne(ctx, colName, filter).Decode(pResult)
}

// TODO selector 使用
func (the *MongoClient) Query(ctx context.Context, colName string, filter any, selector any, pResult any, sortBy ...string) error {
	var cur *mongo.Cursor
	var err error

	if len(sortBy) == 0 {
		cur, err = the.Find(ctx, colName, filter)
	} else {
		var fields []string
		fields = append(fields, sortBy...)
		cur, err = the.Find(ctx, colName, filter, options.Find().SetSort(sort(fields)))
	}

	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	return iterate(ctx, cur, pResult)
}

// QueryCount 符合条件的记录数
func (the *MongoClient) QueryCount(ctx context.Context, colName string, filter any) (int64, error) {
	n, err := the.CountDocuments(ctx, colName, filter)
	return n, err
}

// Insert 新增
func (the *MongoClient) Insert(ctx context.Context, colName string, docs ...any) error {
	var err error
	count := len(docs)
	if count == 1 {
		_, err = the.InsertOne(ctx, colName, docs[0])
	} else if count == 0 {
		return nil
	} else {
		_, err = the.InsertMany(ctx, colName, docs)
	}
	return err
}

// Delete 删除
func (the *MongoClient) Delete(ctx context.Context, colName string, filter any) (int64, error) {
	return the.DeleteMany(ctx, colName, filter)
}

// PagingQuery 分页查询
func (the *MongoClient) QueryPaging(ctx context.Context, colName string, filter any,
	sort any, pageIndex, pageSize int32, pResult any) (total int64, err error) {
	if pageIndex < 1 {
		pageIndex = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	skip := int64((pageIndex - 1) * pageSize)
	limit := int64(pageSize)

	var cur *mongo.Cursor

	if sort == nil {
		cur, err = the.Find(ctx, colName, filter,
			options.Find().SetSkip(skip).SetLimit(limit))
	} else {
		cur, err = the.Find(ctx, colName, filter,
			options.Find().SetSort(sort).SetSkip(skip).SetLimit(limit))
	}

	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	err = cur.All(ctx, pResult)
	if err != nil {
		return 0, err
	}

	count, err := the.QueryCount(ctx, colName, filter)
	if err != nil {
		return 0, err
	}

	return count, err
}

// PagingQuery 分页查询
func (the *MongoClient) PagingQuery(ctx context.Context, colName string, filter any,
	sortBy []string, pageIndex, pageSize int32, pResult any) (int64, error) {

	if pageIndex < 1 {
		pageIndex = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}

	skip := (pageIndex - 1) * pageSize

	var cur *mongo.Cursor
	var err error

	if len(sortBy) == 0 {
		cur, err = the.Find(ctx, colName, filter,
			options.Find().SetSkip(int64(skip)).SetLimit(int64(pageSize)))
	} else {
		cur, err = the.Find(ctx, colName, filter, options.Find().
			SetSort(sort(sortBy)).SetSkip(int64(skip)).SetLimit(int64(pageSize)))
	}

	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	err = cur.All(ctx, pResult)
	if err != nil {
		return 0, err
	}

	count, err := the.QueryCount(ctx, colName, filter)
	if err != nil {
		return 0, err
	}

	return count, err
}

func (the *MongoClient) PagingQueryWithPro(ctx context.Context, colName string, filter interface{},
	sortBy []string, pageIndex, pageSize int32, pResult interface{}, projection interface{}) (int64, error) {

	if pageIndex < 1 {
		pageIndex = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}

	skip := (pageIndex - 1) * pageSize

	var cur *mongo.Cursor
	var err error

	if len(sortBy) == 0 {
		cur, err = the.Find(ctx, colName, filter,
			options.Find().SetProjection(projection).SetSkip(int64(skip)).SetLimit(int64(pageSize)))
	} else {
		cur, err = the.Find(ctx, colName, filter, options.Find().SetProjection(projection).
			SetSort(sort(sortBy)).SetSkip(int64(skip)).SetLimit(int64(pageSize)))
	}

	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	err = cur.All(ctx, pResult)
	if err != nil {
		return 0, err
	}

	count, err := the.QueryCount(ctx, colName, filter)
	if err != nil {
		return 0, err
	}

	return count, err
}

func (the *MongoClient) UpdateAll(ctx context.Context, colName string, filter any, update any, ops any) error {

	var err error

	switch update.(type) {
	case bson.M:
	default:
		update = bson.M{
			"$set": struct2BsonM(update),
		}
	}

	switch ops.(type) {
	case *options.UpdateOptions:
		o := ops.(*options.UpdateOptions)
		_, err = the.UpdateMany(ctx, colName, filter, update, o)
	default:
		_, err = the.UpdateMany(ctx, colName, filter, update)
	}

	return err
}

func (the *MongoClient) UpdateSingleById(ctx context.Context, colName string, id string, update any, noUpdateField ...string) error {
	switch update.(type) {
	case bson.M:
	default:
		update = bson.M{
			"$set": struct2BsonM(update, noUpdateField...),
		}
	}

	r, err := the.UpdateByID(ctx, colName, id, update)
	_ = r
	return err
}

func (the *MongoClient) UpdateSingle(ctx context.Context, colName string, filter any, update any, ops any, ignoreField ...string) error {

	var err error

	switch update.(type) {
	case bson.M:
	default:
		update = bson.M{
			"$set": struct2BsonM(update, ignoreField...),
		}
	}

	switch ops.(type) {
	case *options.UpdateOptions:
		o := ops.(*options.UpdateOptions)
		_, err = the.UpdateOne(ctx, colName, filter, update, o)
	default:
		_, err = the.UpdateOne(ctx, colName, filter, update)
	}

	return err
}

func (the *MongoClient) PipeQuery(ctx context.Context, colName string, pipeline any, result any) error {
	cur, err := the.Collection(colName).Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	return iterate(ctx, cur, result)
}

func (the *MongoClient) Upsert(ctx context.Context, colName string, filter, update any) error {
	switch update.(type) {
	case bson.M:
	default:
		update = bson.M{
			"$set": struct2BsonM(update),
		}
	}
	_, err := the.UpdateOne(ctx, colName, filter, update, options.Update().SetUpsert(true))
	return err
}

// IsExist 判断存在
func (the *MongoClient) IsExist(ctx context.Context, colName string, filter bson.M) (bool, error) {
	n, err := the.QueryCount(ctx, colName, filter)
	return n > 0, err
}

// 事务 session; 自己定义超时时间
func (the *MongoClient) UseSession(ctx context.Context, session func(sessionContext mongo.SessionContext) error) error {
	err := the.c.UseSession(ctx, func(sCtx mongo.SessionContext) error {
		// make sure the ctx of c must be sCtx in session
		if err := sCtx.StartTransaction(); err != nil {
			return err
		}

		return session(sCtx)
	})

	return err
}
