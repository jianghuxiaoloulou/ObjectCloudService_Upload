package model

import (
	"WowjoyProject/ObjectCloudService_Upload/pkg/setting"
	"database/sql"

	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type KeyData struct {
	instance_key                  sql.NullInt64
	jpgfile, dcmfile, ip, virpath sql.NullString
	jpgstatus, dcmstatus          sql.NullInt16
}

func NewDBEngine(databaseSetting *setting.DatabaseSettingS) (*sql.DB, error) {
	db, err := sql.Open(databaseSetting.DBType, databaseSetting.DBConn)
	if err != nil {
		return nil, err
	}
	// 数据库最大连接数
	db.SetMaxOpenConns(databaseSetting.MaxIdleConns)
	db.SetMaxIdleConns(databaseSetting.MaxIdleConns)

	return db, nil
}
