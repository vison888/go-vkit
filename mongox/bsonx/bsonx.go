package bsonx

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 等于
func E(key string, value interface{}) bson.E {
	return bson.E{Key: key, Value: value}
}

// 不等于
func Ne(key string, value interface{}) bson.E {
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$ne", Value: value}}}
}

// 小于
func Lt(key string, value interface{}) bson.E {
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$lt", Value: value}}}
}

// 小于等于
func LtE(key string, value interface{}) bson.E {
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$lte", Value: value}}}
}

// 大于
func Gt(key string, value interface{}) bson.E {
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$gt", Value: value}}}
}

// 大于等于
func GtE(key string, value interface{}) bson.E {
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$gte", Value: value}}}
}

// 正则(包含，如 SQL 中的 like)
// options：i=忽略大小写；m=多行匹配模式；x=忽略非转义的空白字符；s=单行匹配模式；
func Regex(key, value, options string) bson.E {
	return bson.E{Key: key,
		Value: primitive.Regex{Pattern: value, Options: options}}

}

// i=忽略大小写
func RegexI(key, value string) bson.E {
	return bson.E{Key: key,
		Value: primitive.Regex{Pattern: value, Options: "i"}}
}

// i=忽略大小写  s=单行匹配模式
func RegexIS(key, value string) bson.E {
	return bson.E{Key: key,
		Value: primitive.Regex{Pattern: value, Options: "is"}}
}

func In(key string, value ...interface{}) bson.E {
	s := toSlice(value...)
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$in", Value: s}}}
}

func Or(e ...bson.E) bson.E {
	or := bson.A{}
	for _, v := range e {
		or = append(or, bson.D{v})
	}

	return bson.E{Key: "$or", Value: or}
}

func All(key string, value ...interface{}) bson.E {
	s := toSlice(value...)
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$all", Value: s}}}
}

// 判断某个键(属性)是否存在
func Exists(key string, exists bool) bson.E {
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$exists", Value: exists}}}
}

// 某个键(属性)的数据类型
func Type(key string, nType int) bson.E {
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$type", Value: nType}}}
}

func ElemMatch(key string, e ...bson.E) bson.E {
	var m bson.D
	for _, v := range e {
		m = append(m, v)
	}
	return bson.E{Key: key,
		Value: bson.D{bson.E{Key: "$elemMatch", Value: m}}}
}

func ToFilter(filter bson.D) interface{} {
	if len(filter) == 0 {
		// cannot transform type primitive.D to a BSON Document:
		// WriteNull can only write while positioned on a Element or Value but is positioned on a TopLevel
		return filter.Map() // 没有查询条件不转成 bson.M 就会报上面的错误
	}

	return filter
}

// 排序
func Sort(key string, asc bool) bson.D {
	order := -1
	if asc {
		order = 1
	}

	return bson.D{bson.E{Key: key, Value: order}}
}

// 只设置一个属性
func SetKV(key string, value interface{}) bson.D {
	return Set(bson.E{Key: key, Value: value})
}

// 可以设置多个属性，可以用 bsonx.E
func Set(e ...bson.E) bson.D {
	value := bson.D{}
	value = append(value, e...)

	return bson.D{bson.E{Key: "$set", Value: value}}
}

// 增量 increments
func Inc(key string, increment int) bson.E {
	return bson.E{Key: "$inc",
		Value: bson.D{bson.E{Key: key, Value: increment}}}
}

// 将数组中按值或条件移除
func Pull(e ...bson.E) bson.D {
	// 使用
	// Pull(bsonx.E("子对象_数组", bson.M{"子对象_id": 要删除的 id 值}))
	// Pull(bsonx.E("数组", 数组中的值)) //
	value := bson.D{}
	value = append(value, e...)

	return bson.D{bson.E{Key: "$pull", Value: value}}
}
