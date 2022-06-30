package object

import (
	"WowjoyProject/ObjectCloudService_Upload/global"
	"WowjoyProject/ObjectCloudService_Upload/internal/model"
	"WowjoyProject/ObjectCloudService_Upload/pkg/errcode"
	"WowjoyProject/ObjectCloudService_Upload/pkg/general"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"time"
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
	// 判断文件大小，来区别是否开始分段上传
	var code string
	fileSize := general.GetFileSize(obj.FilePath)
	if fileSize >= (int64(global.ObjectSetting.File_Fragment_Size << 20)) {
		// 大文件上传
		code = UploadLargeFile(obj, fileSize)
	} else {
		// 小文件上传
		code = UploadFile(obj)
	}
	if code == "00000" {
		//上传成功更新数据库
		global.Logger.Info("数据上传成功", obj.Key)
		model.UpdateUplaod(obj.Key, obj.Type, obj.FileKey, true)
	} else if code == "A2105" {
		global.Logger.Info("请求限流，重新放入任务队列", obj.Key)
		data := global.ObjectData{
			InstanceKey: obj.Key,
			FileKey:     obj.FileKey,
			FilePath:    obj.FilePath,
			Type:        obj.Type,
			Count:       1,
		}
		global.ObjectDataChan <- data
	} else {
		global.Logger.Info("数据上传失败", obj.Key)
		// 上传失败时先补偿操作，补偿操作失败后才更新数据库
		// if !ReDo(obj) {
		// 	global.Logger.Info("数据补偿失败", obj.Key)
		// 	// 上传失败更新数据库
		// 	model.UpdateUplaod(obj.Key, obj.Type, obj.FileKey, false)
		// }
		model.UpdateUplaod(obj.Key, obj.Type, obj.FileKey, false)
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
		global.Logger.Error("Open File err :", err)
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
		global.Logger.Error("File Copy err :", err)
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
	connectTimeout := 60 * time.Second
	readWriteTimeout := 60 * time.Second
	transport := http.Transport{
		DisableKeepAlives: true,
		// TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		Dial: TimeoutDialer(connectTimeout, readWriteTimeout),
	}
	client := &http.Client{
		Transport: &transport,
	}
	resp, err := client.Do(request)
	global.Logger.Info("开始发起http client.Do: ", obj.Key)
	if err != nil {
		global.Logger.Error("Do Request got err: ", err)
		return errcode.Http_RequestError.Msg()
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		global.Logger.Error("ioutil.ReadAll err: ", err)
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

// // UploadLargeFile 上传大文件
func UploadLargeFile(obj *Object, size int64) string {
	global.Logger.Debug("开始执行大文件上传", obj.Key)
	// num := math.Ceil(float64(size) / float64(global.ObjectSetting.Each_Section_Size))
	// 1.初始化
	UploadId := Multipart_Upload_Init(obj)
	if UploadId == "" {
		global.Logger.Error("分段上传初始化获取UploadId是空,结束任务")
		return ""
	}
	global.Logger.Info("UploadId: ", UploadId)
	// 2.开始上传小段对象
	var uploadRsult []global.FileResult
	var status bool
	var code string
	// 将大文件分成小文件
	TragetSize := global.ObjectSetting.Each_Section_Size << 20
	var fileMap = make(map[int]string)
	fileMap = general.FileSplit(obj.FilePath, int64(TragetSize))
	global.Logger.Debug("文件分段的map: ", fileMap)
	status, uploadRsult = Multipart_Upload(obj, UploadId, fileMap)
	if status {
		// 文件上传成功完结操作
		code = Multipart_Completion(obj, UploadId, uploadRsult)
	} else {
		// 文件上传失败取消操作
		Multipart_Abortion(obj, UploadId)
	}
	// 删除分段文件
	for _, k := range fileMap {
		os.Remove(k)
	}

	return code
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

func TimeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, nil
	}
}

// // 1.文件分段上传初始化
func Multipart_Upload_Init(obj *Object) string {
	global.Logger.Debug("文件分段上传初始化", obj.Key)
	url := global.ObjectSetting.OBJECT_Multipart_Init_URL
	url += "//"
	url += global.ObjectSetting.OBJECT_ResId
	url += "//"
	url += obj.FileKey
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		global.Logger.Error("NewRequest err: ", err, url)
		return errcode.Http_RequestError.Msg()
	}
	// 设置AK
	request.Header.Set("accessKey", global.ObjectSetting.OBJECT_AK)
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	request.Header.Set("Connection", "close")
	connectTimeout := 20 * time.Second
	readWriteTimeout := 20 * time.Second
	transport := http.Transport{
		DisableKeepAlives: true,
		// TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		Dial: TimeoutDialer(connectTimeout, readWriteTimeout),
	}
	client := &http.Client{
		Transport: &transport,
	}
	resp, err := client.Do(request)
	global.Logger.Info("开始发起http client.Do: ", obj.Key)
	if err != nil {
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
		if resultcode != "00000" {
			global.Logger.Error("文件分段上传初始化接口返回错误", resultcode)
			return ""
		}
	}
	if vData, ok := result["data"]; ok {
		dataMap := vData.(map[string]interface{})
		uploadId := dataMap["uploadId"].(string)
		return uploadId
	}
	return ""
}

// // 2.分段对象上传
func Multipart_Upload(obj *Object, uploadid string, fileMap map[int]string) (bool, []global.FileResult) {
	global.Logger.Info(obj.Key, " 开始执行分段上传函数,UploadId: ", uploadid)
	status := true
	size := global.ObjectSetting.Each_Section_Size << 20
	var fileResultList []global.FileResult
	num := len(fileMap)
	for v, k := range fileMap {
		// 讲分段文件多线程上传修改为单线程上传
		var code string
		var index int
		var fileResult global.FileResult
		if status {
			if v == num {
				index, code, fileResult = Multipart_Unifile(obj, k, uploadid, int64(size), v, true)
			} else {
				index, code, fileResult = Multipart_Unifile(obj, k, uploadid, int64(size), v, false)
			}
			if code == "00000" {
				//上传成功更新数据库
				global.Logger.Info(obj.Key, " :的第", index, "段数据上传成功", fileResult)
				fileResultList = append(fileResultList, fileResult)
			} else {
				global.Logger.Info(obj.Key, " :的第", index, "段数据上传失败: ", code)
				// model.UpdateUplaode(obj.InstanceKey, obj.Key, false)
				status = false
			}
		}
	}
	return status, fileResultList
}

// 分段单文件处理
func Multipart_Unifile(obj *Object, filepath string, uploadid string, size int64, num int, flag bool) (int, string, global.FileResult) {
	global.Logger.Debug("文件分段上传单文件: ", obj.Key, " 当前分段：", num)
	var resultdata global.FileResult
	var resultcode string
	url := global.ObjectSetting.OBJECT_Multipart_Upload_URL
	url += "//"
	url += global.ObjectSetting.OBJECT_ResId
	url += "//"
	url += obj.FileKey
	file, err := os.Open(filepath)
	if err != nil {
		return num, errcode.File_OpenError.Msg(), resultdata
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// writer.WriteField("resId", global.ObjectSetting.OBJECT_ResId)
	// writer.WriteField("key", obj.FileKey)
	writer.WriteField("uploadId", uploadid)
	writer.WriteField("filePosition", fmt.Sprintf("%d", int64(num-1)*size))
	writer.WriteField("partNumber", fmt.Sprintf("%d", num))
	if flag {
		writer.WriteField("lastPart", "true")
	}
	formFile, err := writer.CreateFormFile("file", filepath)
	if err != nil {
		global.Logger.Error("CreateFormFile err :", err, file)
		return num, errcode.Http_HeadError.Msg(), resultdata
	}
	_, err = io.Copy(formFile, file)
	if err != nil {
		global.Logger.Error("io.Copy err :", err, file)
		return num, errcode.File_CopyError.Msg(), resultdata
	}
	writer.Close()
	request, err := http.NewRequest("POST", url, body)
	// global.Logger.Debug(body)
	if err != nil {
		global.Logger.Error("NewRequest err: ", err, url)
		return num, errcode.Http_RequestError.Msg(), resultdata
	}
	// request.Header.Set("Authorization", token)
	// 设置AK
	request.Header.Set("accessKey", global.ObjectSetting.OBJECT_AK)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Connection", "close")
	connectTimeout := 20 * time.Second
	readWriteTimeout := 20 * time.Second
	transport := http.Transport{
		DisableKeepAlives: true,
		// TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		Dial: TimeoutDialer(connectTimeout, readWriteTimeout),
	}
	client := &http.Client{
		Transport: &transport,
	}
	resp, err := client.Do(request)
	if err != nil {
		global.Logger.Error("Do Request got err: ", err)
		return num, errcode.Http_RequestError.Msg(), resultdata
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		global.Logger.Error("ioutil.ReadAll got err: ", err)
		return num, errcode.Http_RespError.Msg(), resultdata
	}
	global.Logger.Info("resp.Body: ", string(content))
	var result = make(map[string]interface{})
	err = json.Unmarshal(content, &result)
	if err != nil {
		global.Logger.Error("resp.Body: ", "错误")
		return num, errcode.Http_RespError.Msg(), resultdata
	}
	// 解析json
	if vCode, ok := result["code"]; ok {
		resultcode = vCode.(string)
		if resultcode != "00000" {
			global.Logger.Error("文件分段上传初始化接口返回错误", resultcode)
			return num, resultcode, resultdata
		}
	}
	if vData, ok := result["data"]; ok {
		dataMap := vData.(map[string]interface{})
		global.Logger.Debug(dataMap)
		partNumber := int(dataMap["partNumber"].(float64))
		etag := dataMap["etag"].(string)
		resultdata.PartNumber = partNumber
		resultdata.Etag = etag
		global.Logger.Debug("resultdata: ", resultdata)
	}
	global.Logger.Debug("key: ", obj.Key, "num: ", num, ", resultcode: ", resultcode, ", resultdata", resultdata)
	return num, resultcode, resultdata
}

// 完成对象分段上传
func Multipart_Completion(obj *Object, uploadid string, fileresult []global.FileResult) string {
	global.Logger.Debug("完成对象分段上传: ", obj.Key)
	url := global.ObjectSetting.OBJECT_Multipart_Completion_URL
	url += "//"
	url += global.ObjectSetting.OBJECT_ResId
	url += "//"
	url += obj.FileKey
	jsonData := global.JosnData{
		UploadId:  uploadid,
		PartEtags: fileresult,
	}
	global.Logger.Info(jsonData)

	jsonstr, err := json.Marshal(jsonData)
	if err != nil {
		global.Logger.Error(err)
		return err.Error()
	}
	reader := bytes.NewBuffer(jsonstr)
	global.Logger.Info(string(jsonstr))

	request, err := http.NewRequest("POST", url, reader)
	if err != nil {
		global.Logger.Error("NewRequest err: ", err, url)
		return errcode.Http_RequestError.Msg()
	}
	// 设置AK
	request.Header.Set("accessKey", global.ObjectSetting.OBJECT_AK)
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	request.Header.Set("Connection", "close")
	connectTimeout := 20 * time.Second
	readWriteTimeout := 20 * time.Second
	transport := http.Transport{
		DisableKeepAlives: true,
		// TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		Dial: TimeoutDialer(connectTimeout, readWriteTimeout),
	}
	client := &http.Client{
		Transport: &transport,
	}
	resp, err := client.Do(request)
	if err != nil {
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

// 取消对象分段上传
func Multipart_Abortion(obj *Object, uploadid string) string {
	global.Logger.Debug("取消对象分段上传: ", obj.Key, " Uploadid: ", uploadid)
	url := global.ObjectSetting.OBJECT_Multipart_Abortion_URL
	url += "//"
	url += global.ObjectSetting.OBJECT_ResId
	url += "//"
	url += obj.FileKey
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("uploadId", uploadid)

	writer.Close()
	request, err := http.NewRequest("POST", url, body)
	// global.Logger.Debug(body)
	if err != nil {
		global.Logger.Error("NewRequest err: ", err, url)
		return ""
	}
	// request.Header.Set("Authorization", token)
	// 设置AK
	request.Header.Set("accessKey", global.ObjectSetting.OBJECT_AK)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Connection", "close")
	connectTimeout := 20 * time.Second
	readWriteTimeout := 20 * time.Second
	transport := http.Transport{
		DisableKeepAlives: true,
		// TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		Dial: TimeoutDialer(connectTimeout, readWriteTimeout),
	}
	client := &http.Client{
		Transport: &transport,
	}
	resp, err := client.Do(request)
	if err != nil {
		global.Logger.Error("Do Request got err: ", err)
		return ""
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		global.Logger.Error("ioutil.ReadAll got err: ", err)
		return ""
	}
	global.Logger.Info("取消对象分段上传 resp.Body: ", string(content))
	var result = make(map[string]interface{})
	err = json.Unmarshal(content, &result)
	if err != nil {
		global.Logger.Error("resp.Body: ", "错误")
		return ""
	}
	// 解析json
	if vCode, ok := result["code"]; ok {
		resultcode := vCode.(string)
		global.Logger.Info("resultcode: ", resultcode)
		return resultcode
	}
	return ""
}
