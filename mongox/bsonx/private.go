package bsonx

import "reflect"

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
