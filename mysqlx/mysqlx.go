package mysqlx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/visonlv/go-vkit/logger"
	"gorm.io/gorm"
)

//支持批量插入
func (the *MysqlClient) Insert(o interface{}) error {
	result := the.db.Create(o)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected <= 0 {
		return errors.New("Insert RowsAffected<=0")
	}
	return nil
}

//分批插入 大量数据可用
func (the *MysqlClient) InsertBatch(o interface{}, batchSize int) error {
	result := the.db.CreateInBatches(o, batchSize)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (the *MysqlClient) DeleteById(o interface{}, id string) (bool, error) {
	result := the.db.Delete(o, "id = ?", id)
	if result.Error != nil {
		return false, result.Error
	}
	return true, nil
}

//只能通过id删除
func (the *MysqlClient) DeleteEx(o interface{}) (bool, error) {
	result := the.db.Delete(o)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, result.Error
	}
	if result.RowsAffected <= 0 {
		return false, nil
	}
	return true, nil
}

//通过id查找
func (the *MysqlClient) FindById(o interface{}, id string) (bool, error) {
	result := the.db.First(o, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func (the *MysqlClient) FindPage(page int32, size int32, o interface{}, count *int32) error {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	skip := int64((page - 1) * size)
	limit := int64(size)

	var count64 int64
	result := the.Count(&count64) //总行数
	if result.db.Error != nil {
		return result.db.Error
	}

	*count = int32(count64)
	result = the.Offset(int(skip)).Limit(int(limit)).Find(o) //查询pageindex页的数据

	if result.db.Error != nil {
		return result.db.Error
	}
	return nil
}

func (the *MysqlClient) FindRawPage(sql string, page int32, size int32, o interface{}, count *int32) error {
	// 数量sql
	index := strings.Index(sql, "from")
	if index == -1 {
		return fmt.Errorf("不支持该sql %s", sql)
	}
	countSql := "select count(1) " + sql[index:]

	var count64 int64
	result := the.Raw(countSql).Count(&count64)
	if result.GetDB().Error != nil {
		logger.Errorf("FindRawPage countSql:%s", countSql)
		return result.GetDB().Error
	}
	*count = int32(count64)

	// 分页sql
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	skip := int64((page - 1) * size)
	limit := int64(size)
	pageSql := sql + fmt.Sprintf(" limit %d,%d", skip, limit)
	result = the.Raw(pageSql).Find(o)
	if result.db.Error != nil {
		logger.Errorf("FindRawPage pageSql:%s", pageSql)
		return result.db.Error
	}
	return nil
}

func (the *MysqlClient) FindList(o interface{}) error {
	result := the.Find(o)
	if result.db.Error != nil {
		return result.db.Error
	}
	return nil
}

func (the *MysqlClient) FindFirst(o interface{}) (bool, error) {
	result := the.db.First(o)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

//支持多个数据更新
func (the *MysqlClient) UpdateEx(o interface{}) error {
	result := the.db.Save(o)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

//是否存在
func (the *MysqlClient) Exists(o interface{}) (bool, error) {
	result := the.db.First(o)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}
