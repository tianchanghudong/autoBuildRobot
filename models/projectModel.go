package models

import (
	"autobuildrobot/log"
	"autobuildrobot/tool"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
)

//项目配置
type ProjectModel struct {
	Manager                   string   `json:"Manager"`                   //管理员
	UnopenCommandTypeList     []int    `json:"UnopenCommandTypeList"`     //开放的指令
	ClientEnginePath          string   `json:"ClientEnginePath"`          //客户端引擎地址
	TempBanNormalUserCommands []string `json:"TempBanNormalUserCommands"` //临时禁止普通成员指令数组（如发版本时）
}

var projectFileName = "projectData.gob"
var projectsMap map[string]*ProjectModel
var projectDataLock sync.Mutex

func init() {
	projectsMap = make(map[string]*ProjectModel)
	tool.ReadGobFile(projectFileName, &projectsMap)
}

//有就更新，没有则添加
func UpdateProject(projectName, projectConfig string) (result string, err error) {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()

	//解析数据
	projectModel := new(ProjectModel)
	projectModel.TempBanNormalUserCommands = make([]string, 0)
	projectModel.UnopenCommandTypeList = make([]int, 0)
	err = tool.UnmarshJson([]byte(projectConfig), &projectModel)
	if nil != err {
		return
	}

	//更新或新增
	if _, ok := projectsMap[projectName]; ok {
		//存在则更新
		if "" != projectModel.Manager {
			projectsMap[projectName].Manager = projectModel.Manager
		}
		if "" != projectModel.ClientEnginePath {
			projectsMap[projectName].ClientEnginePath = projectModel.ClientEnginePath
		}
		if nil != projectModel.TempBanNormalUserCommands && len(projectModel.TempBanNormalUserCommands) > 0 {
			projectsMap[projectName].TempBanNormalUserCommands = projectModel.TempBanNormalUserCommands
		}
		if nil != projectModel.UnopenCommandTypeList && len(projectModel.UnopenCommandTypeList) > 0 {
			projectsMap[projectName].UnopenCommandTypeList = projectModel.UnopenCommandTypeList
		}
	} else {
		projectsMap[projectName] = projectModel
	}

	//编码并存储
	tool.SaveGobFile(projectFileName, projectsMap)
	result = "更新项目配置成功"
	return
}

//获取项目配置数据
func GetProjectData(projectName string) (result string) {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()
	if _project, ok := projectsMap[projectName]; ok {
		return tool.MarshalJson(_project)
	}

	//如果不存在项目，则输出默认值
	return "项目不存在，请添加：" + GetProjectConfigHelp()
}

//获取项目配置帮助信息
func GetProjectConfigHelp() (result string) {
	project := new(ProjectModel)
	project.Manager = "项目管理员名字"
	project.ClientEnginePath = "项目客户端引擎（如unity）路径"
	project.UnopenCommandTypeList = make([]int, 0)
	project.TempBanNormalUserCommands = make([]string, 0)
	project.TempBanNormalUserCommands = append(project.TempBanNormalUserCommands, "发版本时禁止成员执行得指令名称，如分支合并")
	return fmt.Sprintf("例：【%s：%s】\n其中，UnopenCommandTypeList是不开放的指令索引数组\n如多个配置用分号分割", commandName[CommandType_UpdateProjectConfig], tool.MarshalJson(project))
}

//获取项目管理员
func GetProjectManager(projectName string) string {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()
	if project, ok := projectsMap[projectName]; ok {
		return project.Manager
	}
	log.Error("项目管理员不存在,请配置！！！")
	return ""
}

//获取项目客户端引擎（Unity）路径
func GetProjectClientEnginePath(projectName string) (error, string) {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()
	if project, ok := projectsMap[projectName]; ok {
		return nil, project.ClientEnginePath
	}
	return errors.New("项目客户端引擎路径失败，项目不存在,请添加！！！"), ""
}

//判断是否为管理员
func JudgeIsManager(projectName, userName string) bool {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()
	if project, ok := projectsMap[projectName]; ok {
		if strings.Contains(project.Manager, userName) {
			return true
		}
	}
	return false
}

//获取不开放的指令  haoxbuxuyao
func GetUnopenCommandList(projectName string) (commandList []int) {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()
	commandList = make([]int, 0)
	if project, ok := projectsMap[projectName]; ok {
		return project.UnopenCommandTypeList
	}
	return commandList
}

//判断指令是否被禁止
func JudgeCommandIsBan(projectName, userName, commandName, svnProject string) bool {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()
	if project, ok := projectsMap[projectName]; ok {
		if strings.Contains(project.Manager, userName) {
			return false
		}
		for _, v := range project.TempBanNormalUserCommands {
			if v == "" {
				continue
			}
			if v == commandName || v == svnProject {
				return true
			}
		}
	}
	return false
}

//将项目名转换成md5串
func ProjectName2Md5(projectName string) string {
	hash := md5.New()
	hash.Write([]byte(projectName))
	cipherText2 := hash.Sum(nil)
	md5Sign := make([]byte, 32)
	hex.Encode(md5Sign, cipherText2)
	return string(md5Sign)
}
