package mysqlx

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/visonlv/go-vkit/logger"
	"github.com/visonlv/go-vkit/utilsx"
)

// CREATE DATABASE IF NOT EXISTS speech DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;
// CREATE TABLE (
// 	`id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'id',
// 	`create_by` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '创建人',
// 	`created_at` timestamp(0) NULL DEFAULT CURRENT_TIMESTAMP(0) COMMENT '创建时间',
// 	`updated_at` timestamp(0) NULL DEFAULT CURRENT_TIMESTAMP(0) COMMENT '更新时间',
// 	`status` smallint(4) NULL DEFAULT 1 COMMENT '状态 0 正常 1 删除',
// 	`password` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '密码',
// 	`username` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT '更新人',
// 	`haha` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL COMMENT 'haha',
// 	PRIMARY KEY (`id`) USING BTREE
// 	) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci ROW_FORMAT = Dynamic;

func getClient() *MysqlClient {
	cc, err := NewClient("root:123456@tcp(127.0.0.1:3306)/mysql?charset=utf8mb4&parseTime=True&loc=Local", 100, 10, 3600)
	if err != nil {
		panic(err)
	}
	cc.db.AutoMigrate(&User{})
	return cc
}
func TestInsert(t *testing.T) {
	cc := getClient()

	user := User{Username: "insert", Password: "haha"}
	user.Id = utilsx.GenUuid()
	cc.Insert(&user)

	user1 := User{Username: "vison1", Password: "haha"}
	user1.Id = utilsx.GenUuid()

	user2 := User{Username: "vison1", Password: "haha"}
	user2.Id = utilsx.GenUuid()

	users := []*User{&user1, &user2}
	cc.Insert(&users)

	time.Sleep(2000)
}
func TestInsertBatch(t *testing.T) {
	cc := getClient()

	user10 := User{Username: "last", Password: "last"}
	user10.Id = utilsx.GenUuid()

	user11 := User{Username: "last", Password: "last"}
	user11.Id = utilsx.GenUuid()

	batchUser := []*User{&user11, &user10}
	cc.InsertBatch(batchUser, 2)

	time.Sleep(2000)
}

func TestDelete(t *testing.T) {
	cc := getClient()

	// u := &User{}
	// u.Password = "haha"
	// cc.Delete(u)

	// u2 := &User{}
	// u2.Id = "3c988d9968b6450c85f8cb90f1deb523"
	// cc.DeleteById( u2, "3c988d9968b6450c85f8cb90f1deb523")

	// cc.Where("password = ?", "haha").Delete(&User{})

	var count int64
	cc.Model(&User{}).Distinct("password").Count(&count)
	logger.Infof("count=:%d", count)
	time.Sleep(2000)
}

func TestUpdate(t *testing.T) {
	cc := getClient()

	u := &User{}
	_, err := cc.FindById(u, "295b95475ca04071912f16fc603e714b")

	if err != nil {
		logger.Infof("FindById fail :%s", err)
		return
	}
	bb, _ := json.Marshal(u)
	logger.Infof("FindById result :%s", string(bb))

	u.Username = "new ======="
	err = cc.UpdateEx(u)
	if err != nil {
		logger.Infof("Update fail :%s", err)
	}

	u1 := &User{}
	cc.FindById(u1, "6e2e71504c4349fd8941f3a7f606dc07")

	u2 := &User{}
	cc.FindById(u2, "3c988d9968b6450c85f8cb90f1deb523")

	u1.Username = "batch -----------------"
	u2.Username = "batch -----------------"
	batchUser := []*User{u1, u2}
	cc.UpdateEx(batchUser)

	// users := []interface{}{u1, u2}
	// cc.UpdateMany( users)

	time.Sleep(2000)
}

func TestDb(t *testing.T) {
}

func TestTransaction(t *testing.T) {
	cc := getClient()

	cc.Transaction(func(tx *MysqlClient) error {
		user10 := User{Username: "TransactionFail", Password: "TransactionFail"}
		user10.Id = utilsx.GenUuid()
		tx.Insert(&user10)
		if true {
			return errors.New("sdsd")
		}
		user11 := User{Username: "TransactionFail", Password: "TransactionFail"}
		user11.Id = utilsx.GenUuid()
		tx.Insert(&user11)
		return nil
	})
}

func TestMysqlClient_Exists(t *testing.T) {
	type args struct {
		o any
	}
	tests := []struct {
		name    string
		the     *MysqlClient
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
		{"测试账号是否存在", getClient().Model(&User{}).Where("password='haha'"), args{&User{}}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.the.Exists(tt.args.o)
			if (err != nil) != tt.wantErr {
				t.Errorf("MysqlClient.Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MysqlClient.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}
