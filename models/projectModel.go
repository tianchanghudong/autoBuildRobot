package models

import (
	"autobuildrobot/log"
	"autobuildrobot/tool"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

//项目配置
type ProjectModel struct {
	Manager                   string   `json:"Manager"`                   //管理员
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
func UpdateProject(projectName, projectConfig string) (result string) {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()

	//解析数据
	projectModel := new(ProjectModel)
	projectModel.TempBanNormalUserCommands = make([]string, 0)
	tool.UnmarshJson([]byte(projectConfig), &projectModel)

	//更新或新增
	if _, ok := projectsMap[projectName]; ok {
		//存在则更新
		if "" != projectModel.Manager {
			projectsMap[projectName].Manager = projectModel.Manager
		}
		if "" != projectModel.ClientEnginePath {
			projectsMap[projectName].ClientEnginePath = projectModel.ClientEnginePath
		}

		//处理禁止指令
		tempBanCommandMap := make(map[string]bool)
		for _, v := range projectsMap[projectName].TempBanNormalUserCommands {
			tempBanCommandMap[v] = true
		}
		for _, v := range projectModel.TempBanNormalUserCommands {
			if strings.Contains(v, "-") {
				delete(tempBanCommandMap, strings.ReplaceAll(v, "-", ""))
				continue
			}
			tempBanCommandMap[v] = true
		}
		newBanNormalUserCommands := make([]string, 0)
		for k, _ := range tempBanCommandMap {
			newBanNormalUserCommands = append(newBanNormalUserCommands, k)
		}
		projectsMap[projectName].TempBanNormalUserCommands = newBanNormalUserCommands
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
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
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
	project.TempBanNormalUserCommands = make([]string, 0)
	project.TempBanNormalUserCommands = append(project.TempBanNormalUserCommands, "如发版本时禁止成员执行得指令名称，如分支合并，如果取消则加上-")
	return fmt.Sprintf("例：【%s：%s】", commandName[CommandType_UpdateProjectConfig], tool.MarshalJson(project))
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
func GetProjectClientEnginePath(projectName string) string {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()
	if project, ok := projectsMap[projectName]; ok {
		return project.ClientEnginePath
	}
	log.Error("项目客户端引擎路径失败，项目不存在,请添加！！！")
	return ""
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

//判断指令是否被禁止
func JudgeCommandIsBan(projectName, userName, commandName string) bool {
	projectDataLock.Lock()
	defer projectDataLock.Unlock()
	if project, ok := projectsMap[projectName]; ok {
		if strings.Contains(project.Manager, userName) {
			return false
		}
		for _, v := range project.TempBanNormalUserCommands {
			if v == commandName {
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
