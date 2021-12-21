package general

import (
	"WowjoyProject/ObjectCloudService_Upload/global"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// 基础函数
// func GetFilePath(file, ip, virpath string) (path string) {
// 	path += "\\\\"
// 	path += ip
// 	path += "\\"
// 	path += virpath
// 	path += "\\"
// 	path += file
// 	return
// }

func GetFilePath(file, ip, virpath string) (key, path string) {

	key += global.ObjectSetting.UPLOAD_ROOT
	key += "\\"
	key += file
	key = strings.Replace(key, "\\", "/", -1)

	path += "\\\\"
	path += ip
	path += "\\"
	path += virpath
	path += "\\"
	path += file
	return
}

func GetFileSize(filename string) int64 {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}

// 检查文件路径
func CheckPath(path string) {
	dir, _ := filepath.Split(path)
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dir, os.ModePerm)
		}
	}
}

// io.copy()来复制
// 参数说明：
// src: 源文件路径
// dest: 目标文件路径
// key :值不为空是更新instance表中的localtion_code值
func CopyFile(src, dest string) (int64, error) {
	// 判断路径文件夹是否存在，不存在，创建文件夹
	CheckPath(dest)
	file1, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer file1.Close()
	file2, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return 0, err
	}
	defer file2.Close()
	return io.Copy(file2, file1)
}

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
