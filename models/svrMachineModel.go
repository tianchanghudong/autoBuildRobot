package models

import (
	"autobuildrobot/log"
	"autobuildrobot/tool"
	"errors"
	"fmt"
	"strings"
	"sync"
)

//游戏服务器主机配置
type SvrMachineModel struct {
	MachineName string `json:"MachineName"` //服务器主机名
	Platform    string `json:"Platform"`    //构建目标平台
	Ip          string `json:"Ip"`
	Port        string `json:"Port"`
	Account     string `json:"Account"`
	Psd         string `json:"Psd"`
	SvrRootPath string `json:"SvrRootPath"`
	ProjectPath string `json:"ProjectPath"` //项目地址
}

var lastSvrMachineConfigFileName string             //上一次的服务器主机配置数据文件名（基本一个项目一个文件）
var svrMachineConfigMap map[string]*SvrMachineModel //服务器主机配置字典，key 服务器主机名 value:服务器主机配置
var svrMachineDataLock sync.Mutex

//有就更新，没有则添加
func UpdateSvrMachineData(projectName, svrConfig string) (result string) {
	svrMachineDataLock.Lock()
	defer svrMachineDataLock.Unlock()

	//先获取数据
	var fileName string
	fileName, svrMachineConfigMap = getProjectSvrMachineData(projectName)

	//再更新数据
	svrArr := strings.Split(svrConfig, ";")
	for _, svr := range svrArr {
		if svr == "" {
			continue
		}
		svrModel := new(SvrMachineModel)
		tool.UnmarshJson([]byte(svr), &svrModel)
		if svrModel.MachineName == "" {
			errMsg := "svr主机名不能为空：" + svr
			log.Error(errMsg)
			result += (errMsg + "\n")
			continue
		}

		//删除配置
		if strings.Contains(svrModel.MachineName, "-") {
			//负号作为删除标记吧
			delBranch := strings.ReplaceAll(svrModel.MachineName, "-", "")
			delete(svrMachineConfigMap, delBranch)
			continue
		}

		//增加或修改
		if _svrModel, ok := svrMachineConfigMap[svrModel.MachineName]; ok {
			//已存在，如果数据为空则用老数据
			if svrModel.Ip == "" {
				svrModel.Ip = _svrModel.Ip
			}
			if svrModel.Platform == "" {
				svrModel.Platform = _svrModel.Platform
			}
			if svrModel.Account == "" {
				svrModel.Account = _svrModel.Account
			}
			if svrModel.Psd == "" || svrModel.Psd == secretFlag {
				svrModel.Psd = _svrModel.Psd
			}
			if svrModel.Port == "" {
				svrModel.Port = _svrModel.Port
			}
			if svrModel.SvrRootPath == "" {
				svrModel.SvrRootPath = _svrModel.SvrRootPath
			}
			if svrModel.ProjectPath == "" {
				svrModel.ProjectPath = _svrModel.ProjectPath
			}
		}
		svrMachineConfigMap[svrModel.MachineName] = svrModel
	}

	//编码并存储
	tool.SaveGobFile(fileName, svrMachineConfigMap)
	if result != "" {
		return
	}
	return "更新主机配置成功"
}

//获取一个项目所有服务器主机配置信息
func QuerySvrMachineDataOfOneProject(projectName, searchValue string) (result string) {
	svrMachineDataLock.Lock()
	defer svrMachineDataLock.Unlock()
	_, svrMachineConfigMap = getProjectSvrMachineData(projectName)
	if len(svrMachineConfigMap) <= 0 {
		return "当前没有主机配置信息，请配置：\n" + GetSvrMachineConfigHelp()
	}

	tpl := SvrMachineModel{}
	for _, v := range svrMachineConfigMap {
		if !JudgeIsSearchAllParam(searchValue) && v.MachineName != searchValue {
			//数据量不大，这里就不再做获取到了退出循环吧
			continue
		}
		tpl.MachineName = v.MachineName
		tpl.Ip = v.Ip
		tpl.Platform = v.Platform
		tpl.Account = v.Account
		tpl.Psd = secretFlag
		tpl.Port = v.Port
		tpl.SvrRootPath = v.SvrRootPath
		tpl.ProjectPath = v.ProjectPath
		result += fmt.Sprintln(tool.MarshalJson(tpl) + "\n")
	}

	if result == "" {
		return "当前没有符合条件的主机配置信息，请配置：\n" + GetSvrMachineConfigHelp()
	} else {
		return "\n***********************以下是主机配置数据***********************\n" + result
	}
}

//获取服务器主机配置帮助提示
func GetSvrMachineConfigHelp() string {
	tpl := SvrMachineModel{
		MachineName: "服务器主机名",
		Platform:    "平台",
		Ip:          "ip",
		Port:        "端口",
		Psd:         "密码",
		Account:     "账号",
		SvrRootPath: "服务器根目录",
		ProjectPath: "工程地址",
	}
	return fmt.Sprintf("例：\n【%s：%s】 \n如多个配置用分号分割", commandName[CommandType_UpdateSvrMachineConfig], tool.MarshalJson(tpl))
}

//获取服务器主机配置数据
func GetSvrMachineData(projectName, svrMachineName string) (err error, ip, port, account, psd, platform, svrRootPath, projectPath string) {
	svrMachineDataLock.Lock()
	defer svrMachineDataLock.Unlock()
	if svrMachineName == "" {
		err = errors.New("获取主机配置失败，主机名不能为空！")
		return
	}
	_, svrMachineConfigMap = getProjectSvrMachineData(projectName)
	if _svrModel, ok := svrMachineConfigMap[svrMachineName]; ok {
		return nil, _svrModel.Ip, _svrModel.Port, _svrModel.Account, _svrModel.Psd, _svrModel.Platform, _svrModel.SvrRootPath, _svrModel.ProjectPath
	} else {
		err = errors.New(svrMachineName + "主机配置不存在，请添加！")
		return
	}
	return
}

//根据项目名获取svr文件名和数据
func getProjectSvrMachineData(projectName string) (string, map[string]*SvrMachineModel) {
	svrDataFileName := "svrMachine.gob"
	fileName := ProjectName2Md5(projectName) + svrDataFileName
	if fileName == lastSvrMachineConfigFileName {
		return fileName, svrMachineConfigMap
	}
	svrMachineConfigMap = make(map[string]*SvrMachineModel)
	tool.ReadGobFile(fileName, &svrMachineConfigMap)
	lastSvrMachineConfigFileName = fileName
	return fileName, svrMachineConfigMap
}
