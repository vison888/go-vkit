package mongox

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func sort(fields []string) bson.D {
	var order bson.D
	for _, field := range fields {
		n := 1
		var kind string
		if field != "" {
			if field[0] == '$' {
				if c := strings.Index(field, ":"); c > 1 && c < len(field)-1 {
					kind = field[1:c]
					field = field[c+1:]
				}
			}
			switch field[0] {
			case '+':
				field = field[1:]
			case '-':
				n = -1
				field = field[1:]
			}
		}
		if field == "" {
			panic("Sort: empty field name")
		}
		if kind == "textScore" {
			order = append(order, bson.E{Key: field, Value: bson.M{"$meta": kind}})
		} else {
			order = append(order, bson.E{Key: field, Value: n})
		}
	}
	return order
}

func iterate(ctx context.Context, cur *mongo.Cursor, result interface{}) error {
	resultv := reflect.ValueOf(result)
	if resultv.Kind() != reflect.Ptr || resultv.Elem().Kind() != reflect.Slice {
		return errors.New("result argument must be a slice address")
	}
	slicev := resultv.Elem()
	slicev = slicev.Slice(0, slicev.Cap())
	element := slicev.Type().Elem()
	i := 0
	for {
		if slicev.Len() == i {
			elemp := reflect.New(element)
			if !cur.Next(ctx) {
				break
			}

			err := cur.Decode(elemp.Interface())
			if err != nil && err != bson.ErrDecodeToNil {
				return err
			}

			slicev = reflect.Append(slicev, elemp.Elem())
			slicev = slicev.Slice(0, slicev.Cap())

		} else {
			if !cur.Next(ctx) {
				break
			}
			err := cur.Decode(slicev.Index(i).Addr().Interface())
			if err != nil && err != bson.ErrDecodeToNil {
				return err
			}
		}
		i++
	}
	resultv.Elem().Set(slicev.Slice(0, i))

	return nil
}

// ignoreField 忽略不要更新的字段，比如 created_on, created_at
func struct2BsonM(obj interface{}, ignoreField ...string) bson.M {
	ignoreField = append(ignoreField, "_id", "created_on", "created_at", "tenant_id")

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		if v.IsNil() {
			panic("nil ptr")
		}
		// 如果是指针，则要判断一下是否为struct
		originType := reflect.ValueOf(obj).Elem().Type()
		if originType.Kind() != reflect.Struct {
			panic("non-struct type")
		}
		// 解引用
		v = v.Elem()
		t = t.Elem()
	}

	var data = make(bson.M)
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Tag.Get("bson")
		name = strings.Replace(name, ",omitempty", "", -1)
		if name == "" {
			name = strings.ToLower(t.Field(i).Name)
		}

		for _, v := range ignoreField {
			if v == name {
				continue
			}
		}
		data[name] = v.Field(i).Interface()
	}
	return data
}

func toSlice(value ...interface{}) []interface{} {
	s := make([]interface{}, 0)

	for _, v := range value {
		vo := reflect.ValueOf(v)
		if vo.Kind() == reflect.Slice {
			for i := 0; i < vo.Len(); i++ {
				s = append(s, vo.Index(i).Interface())
			}
		} else {
			s = append(s, v)
		}
	}

	return s
}
