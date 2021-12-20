package main

import (
	"WowjoyProject/ObjectCloudService_Upload/global"
	"WowjoyProject/ObjectCloudService_Upload/internal/model"
	"WowjoyProject/ObjectCloudService_Upload/pkg/object"
	"WowjoyProject/ObjectCloudService_Upload/pkg/workpattern"

	"github.com/robfig/cron"
)

// @title 本地存储文件上传服务
// @version 1.0.0.1
// @description 存储文件上传
// @termsOfService https://github.com/jianghuxiaoloulou/ObjectCloudService_Upload.git
func main() {
	global.Logger.Info("***开始运行存储策略上传服务***")
	global.ObjectDataChan = make(chan global.ObjectData)
	// 注册工作池，传入任务
	// 参数1 初始化worker(工人)设置最大线程数
	wokerPool := workpattern.NewWorkerPool(global.GeneralSetting.MaxThreads)
	// 有任务就去做，没有就阻塞，任务做不过来也阻塞
	wokerPool.Run()
	// 处理任务
	go func() {
		for {
			select {
			case data := <-global.ObjectDataChan:
				sc := &Dosomething{key: data}
				wokerPool.JobQueue <- sc
			}
		}
	}()
	run()
}

type Dosomething struct {
	key global.ObjectData
}

func (d *Dosomething) Do() {
	global.Logger.Info("正在处理的数据是：", d.key)
	// 处理封装对象操作
	obj := object.NewObject(d.key)
	obj.UploadObject()
}

func run() {
	// 方式一：
	// for {
	// 	time.Sleep(time.Second * 10)
	// 	model.AutoUploadObjectData()
	// }
	// 方式二：获取任务(定时任务)
	MyCron := cron.New()
	MyCron.AddFunc(global.GeneralSetting.CronSpec, func() {
		global.Logger.Info("开始执行定时任务")
		work()
	})
	MyCron.Start()
	defer MyCron.Stop()
	select {}
}

func work() {
	switch global.ObjectSetting.OBJECT_Store_Type {
	case global.PublicCloud:
		global.Logger.Info("***公有云数据上传***")
		model.GetUploadPublicData()
	case global.PrivateCloud:
		global.Logger.Info("***私有云数据上传***")
		model.GetUploadPrivateData()
	}
}
