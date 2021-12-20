package global

import (
	"WowjoyProject/ObjectCloudService_Upload/pkg/logger"
	"WowjoyProject/ObjectCloudService_Upload/pkg/setting"
)

var (
	ServerSetting   *setting.ServerSettingS
	GeneralSetting  *setting.GeneralSettingS
	DatabaseSetting *setting.DatabaseSettingS
	ObjectSetting   *setting.ObjectSettingS
	Logger          *logger.Logger
)
