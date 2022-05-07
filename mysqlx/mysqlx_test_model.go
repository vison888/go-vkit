package mysqlx

import "time"

type BaseModel struct {
	Id        string    `gorm:"primaryKey"`
	CreatedAt time.Time // 在创建时，如果该字段值为零值，则使用当前时间填充
	UpdatedAt time.Time // 在创建时该字段值为零值或者在更新时，使用当前时间戳秒数填充
}

type User struct {
	BaseModel
	Username string `gorm:"column:haha"`
	Password string
}

func (*User) TableName() string {
	return "t_vison_test"
}
