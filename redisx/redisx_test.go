package redisx

import (
	"testing"
)

func TestRedis(t *testing.T) {
	// redis := NewClient("127.0.0.1", "", 1)

	// err := redis.Set(context.Background(), "int30", 30, time.Minute)
	// if err != nil {
	// 	logger.Infof("Set err %s", err.Error())
	// 	return
	// }

	// val, err := redis.GetInt(context.Background(), "int30")
	// if err != nil {
	// 	logger.Infof("err %s", err.Error())
	// 	return
	// }
	// logger.Infof("val %d", val)

	// redis.Set(context.Background(), "inthaha", "haha", time.Minute)
	// val1, err := redis.GetString(context.Background(), "inthaha")
	// if err != nil {
	// 	logger.Infof("err %s", err.Error())
	// 	return
	// }
	// logger.Infof("val %s", val1)

	// type aaa struct {
	// 	Ha string
	// }
	// bb := &aaa{Ha: "obj"}
	// redis.SetJson(context.Background(), "inthaha", bb, time.Minute)

	// cc := &aaa{}
	// err = redis.GetJson(context.Background(), "inthaha", cc)
	// if err != nil {
	// 	logger.Infof("err %s", err.Error())
	// 	return
	// }
	// logger.Infof("val %s", cc.Ha)
}
