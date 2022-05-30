package global

const (
	PublicCloud  int = iota // 共有云
	PrivateCloud            // 私有云
)

// 文件类型
type FileType int

const (
	DCM FileType = iota // DCM 文件
	JPG                 // JPG 文件
)

// 文件状态
type FileStatus int

const (
	FileNotExist FileStatus = iota // 文件不存在
	FileExist                      // 文件存在
	FileFailed                     // 文件失败
)

type ObjectData struct {
	InstanceKey int64    // instance_key 目标key
	FileKey     string   // 文件key
	FilePath    string   // 文件路径
	Type        FileType // 文件类型
	Count       int      // 文件执行次数
}

var (
	ObjectDataChan chan ObjectData
	RunStatus      bool // 当前获取的数据是否运行完成
)

// 分段文件结果
type FileResult struct {
	PartNumber int    `json:"partNumber"`
	Etag       string `json:"etag"`
}

type JosnData struct {
	UploadId  string       `json:"uploadId"`
	PartEtags []FileResult `json:"partETags"`
}
