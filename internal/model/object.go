package model

import (
	"WowjoyProject/ObjectCloudService_Upload/global"
	"WowjoyProject/ObjectCloudService_Upload/pkg/general"
	"time"
)

// 自动上传公有云数据
func GetUploadPublicData() {
	global.Logger.Info("******开始获取自动上传数据******")
	sql := `select ins.instance_key,ins.file_name,im.img_file_name,sl.ip,sl.s_virtual_dir,fr.dcm_file_exist_obs_cloud,fr.img_file_exist_obs_cloud
	from instance ins
	left join image im on im.instance_key = ins.instance_key
	left join file_remote fr on ins.instance_key = fr.instance_key
	left join study_location sl on sl.n_station_code = ins.location_code
	where (fr.dcm_file_exist = 1 and fr.dcm_file_exist_obs_cloud = 0) or (fr.img_file_exist = 1 and fr.img_file_exist_obs_cloud = 0)
	order by ins.instance_key desc limit ?;`
	// global.Logger.Debug(sql)
	rows, err := global.DBEngine.Query(sql, global.GeneralSetting.MaxTasks)
	if err != nil {
		global.Logger.Fatal(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		key := KeyData{}
		_ = rows.Scan(&key.instance_key, &key.dcmfile, &key.jpgfile, &key.ip, &key.virpath, &key.dcmstatus, &key.jpgstatus)
		if key.jpgfile.String != "" && key.jpgstatus.Int16 == int16(global.FileNotExist) {
			fike_key, file_path := general.GetFilePath(key.jpgfile.String, key.ip.String, key.virpath.String)
			global.Logger.Info("需要处理的文件名：", file_path)
			data := global.ObjectData{
				InstanceKey: key.instance_key.Int64,
				FileKey:     fike_key,
				FilePath:    file_path,
				Type:        global.JPG,
				Count:       1,
			}
			global.ObjectDataChan <- data
		}
		if key.dcmfile.String != "" && key.dcmstatus.Int16 == int16(global.FileNotExist) {
			fike_key, file_path := general.GetFilePath(key.dcmfile.String, key.ip.String, key.virpath.String)
			global.Logger.Info("需要处理的文件名：", file_path)
			data := global.ObjectData{
				InstanceKey: key.instance_key.Int64,
				FileKey:     fike_key,
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
	sql := `select ins.instance_key,ins.file_name,im.img_file_name,sl.ip,sl.s_virtual_dir,fr.dcm_file_exist_obs_local,fr.img_file_exist_obs_local
	from instance ins
	left join image im on im.instance_key = ins.instance_key
	left join file_remote fr on ins.instance_key = fr.instance_key
	left join study_location sl on sl.n_station_code = ins.location_code
	where (fr.dcm_file_exist = 1 and fr.dcm_file_exist_obs_local = 0) or (fr.img_file_exist = 1 and fr.img_file_exist_obs_local = 0)
	order by ins.instance_key desc limit ?;`
	// global.Logger.Debug(sql)
	rows, err := global.DBEngine.Query(sql, global.GeneralSetting.MaxTasks)
	if err != nil {
		global.Logger.Fatal(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		key := KeyData{}
		_ = rows.Scan(&key.instance_key, &key.dcmfile, &key.jpgfile, &key.ip, &key.virpath, &key.dcmstatus, &key.jpgstatus)
		if key.jpgfile.String != "" && key.jpgstatus.Int16 == int16(global.FileNotExist) {
			fike_key, file_path := general.GetFilePath(key.jpgfile.String, key.ip.String, key.virpath.String)
			global.Logger.Info("需要处理的文件名：", file_path)
			data := global.ObjectData{
				InstanceKey: key.instance_key.Int64,
				FileKey:     fike_key,
				FilePath:    file_path,
				Type:        global.JPG,
				Count:       1,
			}
			global.ObjectDataChan <- data
		}
		if key.dcmfile.String != "" && key.dcmstatus.Int16 == int16(global.FileNotExist) {
			fike_key, file_path := general.GetFilePath(key.dcmfile.String, key.ip.String, key.virpath.String)
			global.Logger.Info("需要处理的文件名：", file_path)
			data := global.ObjectData{
				InstanceKey: key.instance_key.Int64,
				FileKey:     fike_key,
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
	sql := `update file_remote fr set fr.dcm_file_exist = 1 where fr.instance_key = ?;`
	global.DBEngine.Exec(sql, key)
}

// 上传数据后更新数据库
func UpdateUplaod(key int64, filetype global.FileType, remotekey string, status bool) {
	// 获取更新时时间
	local, _ := time.LoadLocation("Local")
	timeFormat := "2006-01-02 15:04:05"
	curtime := time.Now().In(local).Format(timeFormat)
	switch global.ObjectSetting.OBJECT_Store_Type {
	case global.PublicCloud:
		switch filetype {
		case global.DCM:
			if status {
				global.Logger.Info("***公有云DCM数据上传成功，更新状态***")
				sql := `update file_remote fr set fr.dcm_file_exist_obs_cloud = 1,fr.dcm_location_code_obs_cloud = ?,fr.dcm_update_time_obs_cloud = ?,fr.dcm_file_name_remote = ? where fr.instance_key = ?;`
				global.DBEngine.Exec(sql, global.ObjectSetting.OBJECT_Upload_Success_Code, curtime, remotekey, key)
			} else {
				global.Logger.Info("***公有云DCM数据上传失败，更新状态***")
				sql := `update file_remote fr set fr.dcm_file_exist_obs_cloud = 2 where fr.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			}
		case global.JPG:
			if status {
				global.Logger.Info("***公有云JPG数据上传成功，更新状态***")
				sql := `update file_remote fr set fr.img_file_exist_obs_cloud = 1,fr.img_update_time_obs_cloud = ?,fr.img_file_name_remote=? where fr.instance_key = ?;`
				global.DBEngine.Exec(sql, curtime, remotekey, key)
			} else {
				global.Logger.Info("***公有云JPG数据上传失败，更新状态***")
				sql := `update file_remote fr set fr.img_file_exist_obs_cloud = 2 where fr.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			}
		}
	case global.PrivateCloud:
		switch filetype {
		case global.DCM:
			if status {
				global.Logger.Info("***私有云DCM数据上传成功，更新状态***")
				sql := `update file_remote fr set fr.dcm_file_exist_obs_local = 1,fr.dcm_location_code_obs_local = ?,fr.dcm_update_time_obs_local = ?,fr.dcm_file_name_remote = ? where fr.instance_key = ?;`
				global.DBEngine.Exec(sql, global.ObjectSetting.OBJECT_Upload_Success_Code, curtime, remotekey, key)
			} else {
				global.Logger.Info("***私有云DCM数据上传失败，更新状态***")
				sql := `update file_remote fr set fr.dcm_file_exist_obs_local = 2 where fr.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			}
		case global.JPG:
			if status {
				global.Logger.Info("***私有云JPG数据上传成功，更新状态***")
				sql := `update file_remote fr set fr.img_file_exist_obs_local = 1,fr.img_update_time_obs_local = ?,fr.img_file_name_remote=? where fr.instance_key = ?;`
				global.DBEngine.Exec(sql, curtime, remotekey, key)
			} else {
				global.Logger.Info("***私有云JPG数据上传失败，更新状态***")
				sql := `update file_remote fr set fr.img_file_exist_obs_local = 2 where fr.instance_key = ?;`
				global.DBEngine.Exec(sql, key)
			}
		}
	}
}
