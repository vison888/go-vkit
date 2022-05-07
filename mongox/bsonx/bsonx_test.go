package bsonx

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func Test_In(t *testing.T) {
	x := "x"
	one := bson.E{Key: x, Value: bson.D{bson.E{Key: "$in", Value: bson.A{1}}}}

	t1 := In(x, 1)

	s1 := fmt.Sprintf("%+v", one)
	s2 := fmt.Sprintf("%+v", t1)

	if s1 != s2 {
		println(s1)
		println(s2)
		t.Error()
	}

	three := bson.E{Key: x, Value: bson.D{bson.E{Key: "$in", Value: bson.A{1, 2, 3}}}}
	t2 := In(x, 1, 2, 3)

	s1 = fmt.Sprintf("%+v", three)
	s2 = fmt.Sprintf("%+v", t2)

	if s1 != s2 {
		println(s1)
		println(s2)
		t.Error()
	}

	t3 := In(x, []int{1, 2}, 3)
	s1 = fmt.Sprintf("%+v", three)
	s2 = fmt.Sprintf("%+v", t3)
	if s1 != s2 {
		println(s1)
		println(s2)
		t.Error()
	}

	t4 := In(x, 1, []int{2, 3})
	s1 = fmt.Sprintf("%+v", three)
	s2 = fmt.Sprintf("%+v", t4)
	if s1 != s2 {
		println(s1)
		println(s2)
		t.Error()
	}

}
