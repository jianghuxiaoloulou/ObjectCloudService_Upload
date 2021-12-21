package object

import (
	"WowjoyProject/ObjectCloudService_Upload/global"
	"WowjoyProject/ObjectCloudService_Upload/internal/model"
	"WowjoyProject/ObjectCloudService_Upload/pkg/errcode"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

//var token string

// 封装对象相关操作
type Object struct {
	Key      int64           // 目标key
	FileKey  string          // 文件key
	FilePath string          // 文件路径
	Type     global.FileType // 文件类型
	Count    int             // 文件执行次数
}

func NewObject(data global.ObjectData) *Object {
	return &Object{
		Key:      data.InstanceKey,
		FileKey:  data.FileKey,
		FilePath: data.FilePath,
		Type:     data.Type,
		Count:    data.Count,
	}
}

// 上传对象[POST]
func (obj *Object) UploadObject() {
	global.Logger.Info("开始上传对象：", *obj)
	code := UploadFile(obj)
	if code == "00000" {
		//上传成功更新数据库
		global.Logger.Info("数据上传成功", obj.Key)
		model.UpdateUplaod(obj.Key, obj.Type, obj.FileKey, true)
	} else {
		global.Logger.Info("数据上传失败", obj.Key)
		// 上传失败时先补偿操作，补偿操作失败后才更新数据库
		if !ReDo(obj) {
			global.Logger.Info("数据补偿失败", obj.Key)
			global.Logger.Info(obj.Key, " :文件删除失败，更新标志")
			// 上传失败更新数据库
			model.UpdateUplaod(obj.Key, obj.Type, obj.FileKey, false)
		}
	}
}

// UploadFile 上传文件
func UploadFile(obj *Object) string {
	global.Logger.Debug("开始执行文件上传")
	url := global.ObjectSetting.OBJECT_POST_Upload
	url += "//"
	url += global.ObjectSetting.OBJECT_ResId
	url += "//"
	url += obj.FileKey
	global.Logger.Debug("操作的URL: ", url)
	file, err := os.Open(obj.FilePath)
	if err != nil {
		return errcode.File_OpenError.Msg()
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	formFile, err := writer.CreateFormFile("file", obj.FilePath)
	if err != nil {
		global.Logger.Error("CreateFormFile err :", err, file)
		return errcode.Http_HeadError.Msg()
	}
	_, err = io.Copy(formFile, file)
	if err != nil {
		return errcode.File_CopyError.Msg()
	}

	writer.Close()
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		global.Logger.Error("NewRequest err: ", err, url)
		return errcode.Http_RequestError.Msg()
	}
	// 设置AK
	request.Header.Set("accessKey", global.ObjectSetting.OBJECT_AK)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Connection", "close")
	transport := http.Transport{
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Transport: &transport,
	}
	resp, err := client.Do(request)
	if err != nil {
		// token = ""
		global.Logger.Error("Do Request got err: ", err)
		return errcode.Http_RequestError.Msg()
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errcode.Http_RespError.Msg()
	}
	global.Logger.Info("resp.Body: ", string(content))
	var result = make(map[string]interface{})
	err = json.Unmarshal(content, &result)
	if err != nil {
		global.Logger.Error("resp.Body: ", "错误")
		return errcode.Http_RespError.Msg()
	}
	// 解析json
	if vCode, ok := result["code"]; ok {
		resultcode := vCode.(string)
		global.Logger.Info("resultcode: ", resultcode)
		return resultcode
	}
	return ""
}

// 补偿操作
func ReDo(obj *Object) bool {
	global.Logger.Info("开始补偿操作：", obj.Key)
	if obj.Count < global.ObjectSetting.OBJECT_Count {
		obj.Count += 1
		data := global.ObjectData{
			InstanceKey: obj.Key,
			FileKey:     obj.FileKey,
			FilePath:    obj.FilePath,
			Type:        obj.Type,
			Count:       obj.Count,
		}
		global.ObjectDataChan <- data
		return true
	}
	return false
}
