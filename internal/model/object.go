package model

import (
	"WowjoyProject/ObjectCloudService_Upload/global"
	"WowjoyProject/ObjectCloudService_Upload/pkg/general"
)

// 自动上传公有云数据
func GetUploadPublicData() {
	if global.RunStatus {
		global.Logger.Info("上次获取的数据没有消耗完，等待消耗完，再获取数据....")
		return
	}
	global.RunStatus = true
	global.Logger.Info("******自动上传公有云数据******")
	DICOMData()
	JPGData()
	global.RunStatus = false
}

//自动上传私有云数据
func GetUploadPrivateData() {
	if global.RunStatus {
		global.Logger.Info("上次获取的数据没有消耗完，等待消耗完，再获取数据....")
		return
	}
	global.Logger.Info("******自动上传私有云数据******")
	global.RunStatus = true
	DICOMData()
	JPGData()
	global.RunStatus = false
}

func DICOMData() {
	sql := ""
	switch global.ObjectSetting.OBJECT_Store_Type {
	case global.PublicCloud:
		sql = `SELECT fr.instance_key,ins.file_name,sl.ip,sl.s_virtual_dir
		FROM file_remote fr
		LEFT JOIN instance ins on fr.instance_key = ins.instance_key
		LEFT JOIN study_location sl on sl.n_station_code = ins.location_code
		WHERE 1=1
		AND fr.instance_key >= ?
		AND fr.dcm_file_exist_obs_cloud = 0
		AND fr.dcm_file_exist = 1 
		AND timestampdiff(YEAR,fr.dcm_update_time_retrieve,now()) <= ?
		AND ins.is_del = 0
		ORDER BY fr.dcm_update_time_retrieve DESC
		LIMIT ?;`
	case global.PrivateCloud:
		sql = `SELECT fr.instance_key,ins.file_name,sl.ip,sl.s_virtual_dir
		FROM file_remote fr
		LEFT JOIN instance ins on fr.instance_key = ins.instance_key
		LEFT JOIN study_location sl on sl.n_station_code = ins.location_code
		WHERE 1=1
		AND fr.instance_key >= ?
		AND fr.dcm_file_exist_obs_local = 0
		AND fr.dcm_file_exist = 1 
		AND timestampdiff(YEAR,fr.dcm_update_time_retrieve,now()) <= ?
		AND ins.is_del = 0
		ORDER BY fr.dcm_update_time_retrieve DESC
		LIMIT ?;`
	}
	if global.ReadDBEngine.Ping() != nil {
		global.Logger.Error("ReadDBEngine.ping() err: ", global.ReadDBEngine.Ping())
		global.RunStatus = false
		return
	}
	rows, err := global.ReadDBEngine.Query(sql, global.ObjectSetting.OBJECT_START_KEY, global.ObjectSetting.OBJECT_TIME, global.GeneralSetting.MaxTasks)
	if err != nil {
		global.Logger.Fatal(err)
		global.RunStatus = false
		return
	}
	defer rows.Close()
	for rows.Next() {
		key := KeyData{}
		err = rows.Scan(&key.instance_key, &key.dcmfile, &key.ip, &key.virpath)
		if err != nil {
			global.Logger.Fatal("rows.Scan error: ", err)
			global.RunStatus = false
			return
		}
		if key.dcmfile.String != "" {
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
		}
	}
}

func JPGData() {
	sql := ""
	switch global.ObjectSetting.OBJECT_Store_Type {
	case global.PublicCloud:
		sql = `SELECT fr.instance_key,im.img_file_name,sl.ip,sl.s_virtual_dir
		FROM file_remote fr
		LEFT JOIN instance ins on fr.instance_key = ins.instance_key
		LEFT JOIN image im ON im.instance_key = fr.instance_key
		LEFT JOIN study_location sl on sl.n_station_code = ins.location_code
		WHERE 1=1
		AND fr.instance_key >= ?
		AND fr.img_file_exist_obs_cloud = 0
		AND fr.img_file_exist = 1
		AND timestampdiff(YEAR,fr.dcm_update_time_retrieve,now()) <= ?
		ORDER BY fr.dcm_update_time_retrieve DESC
		LIMIT ?;`
	case global.PrivateCloud:
		sql = `SELECT fr.instance_key,im.img_file_name,sl.ip,sl.s_virtual_dir
		FROM file_remote fr
		LEFT JOIN instance ins on fr.instance_key = ins.instance_key
		LEFT JOIN image im ON im.instance_key = fr.instance_key
		LEFT JOIN study_location sl on sl.n_station_code = ins.location_code
		WHERE 1=1
		AND fr.instance_key >= ?
		AND fr.img_file_exist_obs_local = 0
		AND fr.img_file_exist = 1
		AND timestampdiff(YEAR,fr.dcm_update_time_retrieve,now()) <= ?
		ORDER BY fr.dcm_update_time_retrieve DESC
		LIMIT ?;`
	}
	if global.ReadDBEngine.Ping() != nil {
		global.Logger.Error("ReadDBEngine.ping() err: ", global.ReadDBEngine.Ping())
		global.RunStatus = false
		return
	}
	rows, err := global.ReadDBEngine.Query(sql, global.ObjectSetting.OBJECT_START_KEY, global.ObjectSetting.OBJECT_TIME, global.GeneralSetting.MaxTasks)
	if err != nil {
		global.Logger.Fatal(err)
		global.RunStatus = false
		return
	}
	defer rows.Close()
	for rows.Next() {
		key := KeyData{}
		err = rows.Scan(&key.instance_key, &key.jpgfile, &key.ip, &key.virpath)
		if err != nil {
			global.Logger.Fatal("rows.Scan error: ", err)
			global.RunStatus = false
			return
		}
		if key.jpgfile.String != "" {
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
	}
}

// 更新不存在的DCM字段
func UpdateLocalStatus(key int64) {
	sql := `update file_remote fr set fr.dcm_file_exist = 0 where fr.instance_key = ?;`
	if global.WriteDBEngine.Ping() != nil {
		global.Logger.Error("WriteDBEngine.ping() err: ", global.ReadDBEngine.Ping())
		return
	}
	global.WriteDBEngine.Exec(sql, key)
}

// 更新不存在的JPG字段
func UpdateLocalJPGStatus(key int64) {
	sql := `update file_remote fr set fr.img_file_exist = 0 where fr.instance_key = ?;`
	if global.WriteDBEngine.Ping() != nil {
		global.Logger.Error("WriteDBEngine.ping() err: ", global.ReadDBEngine.Ping())
		return
	}
	global.WriteDBEngine.Exec(sql, key)
}

// 上传数据后更新数据库
func UpdateUplaod(key int64, filetype global.FileType, remotekey string, status bool) {
	// 获取更新时时间
	if global.WriteDBEngine.Ping() != nil {
		global.Logger.Error("WriteDBEngine.ping() err: ", global.ReadDBEngine.Ping())
		return
	}
	switch global.ObjectSetting.OBJECT_Store_Type {
	case global.PublicCloud:
		switch filetype {
		case global.DCM:
			if status {
				global.Logger.Info("***公有云DCM数据上传成功，更新状态***")
				sql := `update file_remote fr set fr.dcm_file_exist_obs_cloud = 1,fr.dcm_location_code_obs_cloud = ?,fr.dcm_update_time_obs_cloud = now(),fr.dcm_file_name_remote = ? where fr.instance_key = ?;`
				global.WriteDBEngine.Exec(sql, global.ObjectSetting.OBJECT_Upload_Success_Code, remotekey, key)
			} else {
				global.Logger.Info("***公有云DCM数据上传失败，更新状态***")
				sql := `update file_remote fr set fr.dcm_file_exist_obs_cloud = 2 where fr.instance_key = ?;`
				global.WriteDBEngine.Exec(sql, key)
			}
		case global.JPG:
			if status {
				global.Logger.Info("***公有云JPG数据上传成功，更新状态***")
				sql := `update file_remote fr set fr.img_file_exist_obs_cloud = 1,fr.img_update_time_obs_cloud = now(),fr.img_file_name_remote=? where fr.instance_key = ?;`
				global.WriteDBEngine.Exec(sql, remotekey, key)
			} else {
				global.Logger.Info("***公有云JPG数据上传失败，更新状态***")
				sql := `update file_remote fr set fr.img_file_exist_obs_cloud = 2 where fr.instance_key = ?;`
				global.WriteDBEngine.Exec(sql, key)
			}
		}
	case global.PrivateCloud:
		switch filetype {
		case global.DCM:
			if status {
				global.Logger.Info("***私有云DCM数据上传成功，更新状态***")
				sql := `update file_remote fr set fr.dcm_file_exist_obs_local = 1,fr.dcm_location_code_obs_local = ?,fr.dcm_update_time_obs_local = now(),fr.dcm_file_name_remote = ? where fr.instance_key = ?;`
				global.WriteDBEngine.Exec(sql, global.ObjectSetting.OBJECT_Upload_Success_Code, remotekey, key)
			} else {
				global.Logger.Info("***私有云DCM数据上传失败，更新状态***")
				sql := `update file_remote fr set fr.dcm_file_exist_obs_local = 2 where fr.instance_key = ?;`
				global.WriteDBEngine.Exec(sql, key)
			}
		case global.JPG:
			if status {
				global.Logger.Info("***私有云JPG数据上传成功，更新状态***")
				sql := `update file_remote fr set fr.img_file_exist_obs_local = 1,fr.img_update_time_obs_local = now(),fr.img_file_name_remote=? where fr.instance_key = ?;`
				global.WriteDBEngine.Exec(sql, remotekey, key)
			} else {
				global.Logger.Info("***私有云JPG数据上传失败，更新状态***")
				sql := `update file_remote fr set fr.img_file_exist_obs_local = 2 where fr.instance_key = ?;`
				global.WriteDBEngine.Exec(sql, key)
			}
		}
	}
}
