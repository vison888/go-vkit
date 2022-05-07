package mysqlx

import (
	"errors"
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
	if result.RowsAffected <= 0 {
		return errors.New("InsertMany RowsAffected<=0")
	}
	return nil
}

func (the *MysqlClient) DeleteById(o interface{}, id string) error {
	result := the.db.Delete(o, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected <= 0 {
		return errors.New("DeleteById RowsAffected<=0")
	}
	return nil
}

//只能通过id删除
func (the *MysqlClient) Delete(o interface{}) error {
	result := the.db.Delete(o)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected <= 0 {
		return errors.New("Delete RowsAffected<=0")
	}
	return nil
}

//通过id查找
func (the *MysqlClient) FindById(o interface{}, id string) error {
	result := the.db.First(o, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected <= 0 {
		return errors.New("empty")
	}
	return nil
}

//支持多个数据更新
func (the *MysqlClient) Update(o interface{}) error {
	result := the.db.Save(o)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
