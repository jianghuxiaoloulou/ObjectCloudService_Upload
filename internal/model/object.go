package model

import (
	"WowjoyProject/ObjectCloudService_Upload/global"
	"WowjoyProject/ObjectCloudService_Upload/pkg/general"
)

// 自动上传公有云数据
func GetUploadPublicData() {
	global.Logger.Info("******开始获取自动上传数据******")
	sql := `select ins.instance_key,ins.file_name,im.img_file_name,sl.ip,sl.s_virtual_dir
	from instance ins
	left join image im on im.instance_key = ins.instance_key
	left join study_location sl on sl.n_station_code = ins.location_code
	where ins.FileExist = 1 and ins.file_exist_obs_cloud = 0
	order by ins.instance_key asc limit ?;`
	// global.Logger.Debug(sql)
	rows, err := global.DBEngine.Query(sql, global.GeneralSetting.MaxTasks)
	if err != nil {
		global.Logger.Fatal(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		key := KeyData{}
		_ = rows.Scan(&key.instance_key, &key.dcmfile, &key.jpgfile, &key.ip, &key.virpath)
		if key.jpgfile.String != "" {
			file_path := general.GetFilePath(key.jpgfile.String, key.ip.String, key.virpath.String)
			global.Logger.Info("需要处理的文件名：", file_path)
			data := global.ObjectData{
				InstanceKey: key.instance_key.Int64,
				FileKey:     key.jpgfile.String,
				FilePath:    file_path,
				Type:        global.JPG,
				Count:       1,
			}
			global.ObjectDataChan <- data
		}
		if key.dcmfile.String != "" {
			file_path := general.GetFilePath(key.dcmfile.String, key.ip.String, key.virpath.String)
			global.Logger.Info("需要处理的文件名：", file_path)
			data := global.ObjectData{
				InstanceKey: key.instance_key.Int64,
				FileKey:     key.dcmfile.String,
				FilePath:    file_path,
				Type:        global.DCM,
				Count:       1,
			}
			global.ObjectDataChan <- data
		} else {
			global.Logger.Error(key.instance_key.Int64, ": DCM文件不存在")
			UpdateLocalStatus(key.instance_key.Int64)
		}
	}
}

//自动上传私有云数据
func GetUploadPrivateData() {
	global.Logger.Info("******开始获取自动上传数据******")
	sql := `select ins.instance_key,ins.file_name,im.img_file_name,sl.ip,sl.s_virtual_dir
	from instance ins
	left join image im on im.instance_key = ins.instance_key
	left join study_location sl on sl.n_station_code = ins.location_code
	where ins.FileExist = 1 and ins.file_exist_ibs_local = 0
	order by ins.instance_key asc limit ?;`
	// global.Logger.Debug(sql)
	rows, err := global.DBEngine.Query(sql, global.GeneralSetting.MaxTasks)
	if err != nil {
		global.Logger.Fatal(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		key := KeyData{}
		_ = rows.Scan(&key.instance_key, &key.dcmfile, &key.jpgfile, &key.ip, &key.virpath)
		if key.jpgfile.String != "" {
			file_path := general.GetFilePath(key.jpgfile.String, key.ip.String, key.virpath.String)
			global.Logger.Info("需要处理的文件名：", file_path)
			data := global.ObjectData{
				InstanceKey: key.instance_key.Int64,
				FileKey:     key.jpgfile.String,
				FilePath:    file_path,
				Type:        global.JPG,
				Count:       1,
			}
			global.ObjectDataChan <- data
		}
		if key.dcmfile.String != "" {
			file_path := general.GetFilePath(key.dcmfile.String, key.ip.String, key.virpath.String)
			global.Logger.Info("需要处理的文件名：", file_path)
			data := global.ObjectData{
				InstanceKey: key.instance_key.Int64,
				FileKey:     key.dcmfile.String,
				FilePath:    file_path,
				Type:        global.DCM,
				Count:       1,
			}
			global.ObjectDataChan <- data
		} else {
			global.Logger.Error(key.instance_key.Int64, ": DCM文件不存在")
			UpdateLocalStatus(key.instance_key.Int64)
		}
	}
}

// 更新不存在的DCM字段
func UpdateLocalStatus(key int64) {
	global.Logger.Info("***DCM文件不存在，更新状态***")
	sql := `update instance ins set ins.FileExist = 0 where ins.instance_key = ?;`
	global.DBEngine.Exec(sql, key)
}

// 上传数据后更新数据库
func UpdateUplaod(key int64, filetype global.FileType, status bool) {
	switch global.ObjectSetting.OBJECT_Store_Type {
	case global.PublicCloud:
		switch filetype {
		case global.DCM:
			if status {
				global.Logger.Info("***公有云DCM数据上传成功，更新状态***")
				sql := `update instance ins set ins.file_exist_obs_cloud = 1 where ins.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			} else {
				global.Logger.Info("***公有云DCM数据上传失败，更新状态***")
				sql := `update instance ins set ins.file_exist_obs_cloud = 2 where ins.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			}
		case global.JPG:
			if status {
				global.Logger.Info("***公有云JPG数据上传成功，更新状态***")
				sql := `update image im set im.file_exist_obs_cloud = 1 where im.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			} else {
				global.Logger.Info("***公有云JPG数据上传失败，更新状态***")
				sql := `update image im set im.file_exist_obs_cloud = 2 where im.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			}
		}
	case global.PrivateCloud:
		switch filetype {
		case global.DCM:
			if status {
				global.Logger.Info("***私有云DCM数据上传成功，更新状态***")
				sql := `update instance ins set ins.file_exist_obs_local = 1 where ins.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			} else {
				global.Logger.Info("***私有云DCM数据上传失败，更新状态***")
				sql := `update instance ins set ins.file_exist_obs_local = 2 where ins.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			}
		case global.JPG:
			if status {
				global.Logger.Info("***私有云JPG数据上传成功，更新状态***")
				sql := `update image im set im.file_exist_obs_local = 1 where im.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			} else {
				global.Logger.Info("***私有云JPG数据上传失败，更新状态***")
				sql := `update image im set im.file_exist_obs_local = 2 where im.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			}
		}
	}
}