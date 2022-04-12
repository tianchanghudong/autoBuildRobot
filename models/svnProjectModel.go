package models

import (
	"autobuildrobot/log"
	"autobuildrobot/tool"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

//svn工程，区别于ProjectModel，一个ProjectModel对应多个SvnProjectModel
type SvnProjectModel struct {
	ProjectName              string   `json:"ProjectName"`              //工程名称
	ProjectPath              string   `json:"ProjectPath"`              //工程地址
	SvnUrl                   string   `json:"SvnUrl"`                   //svn地址
	ConflictAutoWayWhenMerge string   `json:"ConflictAutoWayWhenMerge"` //合并冲突处理方式
	LastGetSvnLogTime        int64    `json:"-"`                        //上次获取svn日志时间
	AutoBuildMethodList      []string `json:"AutoBuildMethodList"`      //项目自动构建方法列表
}

var lastProjectFileName string                //上一个项目的svn工程数据文件名（基本一个项目一个svn工程数据文件）
var svnProjectMap map[string]*SvnProjectModel //项目分支配置字典，key 分支名 value:项目分支
var mergeFlags = []string{"合并到", "合并"}        //项目合并标识，按顺序分割获取两个分支
var svnProjectDataLock sync.Mutex

//有就更新，没有则添加
func UpdateSvnProject(projectName, svnProjectConfig string) (result string) {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()

	//先获取数据
	var fileName string
	fileName, svnProjectMap = getSvnProjectsDataByProjectName(projectName)

	//再更新数据
	avnProjectArr := strings.Split(svnProjectConfig, ";")
	for _, svnProject := range avnProjectArr {
		if svnProject == "" {
			continue
		}
		svnProjectModel := new(SvnProjectModel)
		tool.UnmarshJson([]byte(svnProject), &svnProjectModel)
		if svnProjectModel.ProjectName == "" {
			log.Error("svn工程名不能为空：", svnProject)
			continue
		}
		if strings.Contains(svnProjectModel.ProjectName, "-") {
			//负号作为删除标记吧
			delSvnProject := strings.ReplaceAll(svnProjectModel.ProjectName, "-", "")
			delete(svnProjectMap, delSvnProject)
			continue
		}
		if oldSvnProject, ok := svnProjectMap[svnProjectModel.ProjectName]; ok {
			//存在则LastGetSvnLogTime不能被修改
			svnProjectModel.LastGetSvnLogTime = oldSvnProject.LastGetSvnLogTime

			//如果没有数据则用老的赋值
			if svnProjectModel.ProjectPath == "" {
				svnProjectModel.ProjectPath = oldSvnProject.ProjectPath
			}
			if svnProjectModel.SvnUrl == "" {
				svnProjectModel.SvnUrl = oldSvnProject.SvnUrl
			}
			if svnProjectModel.ConflictAutoWayWhenMerge == "" {
				svnProjectModel.ConflictAutoWayWhenMerge = oldSvnProject.ConflictAutoWayWhenMerge
			}
			if nil == svnProjectModel.AutoBuildMethodList || len(svnProjectModel.AutoBuildMethodList) <= 0 {
				//不要那么复杂了，就直接用新得替换，只能更新整个
				svnProjectModel.AutoBuildMethodList = oldSvnProject.AutoBuildMethodList
			}
		}
		svnProjectMap[svnProjectModel.ProjectName] = svnProjectModel
	}

	//编码并存储
	tool.SaveGobFile(fileName, svnProjectMap)
	result = "更新svn工程配置成功"
	return
}

//获取一个项目所有分支配置信息
func QuerySvnProjectsDataByProject(projectName, searchValue string) (result string) {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()

	_, svnProjectMap = getSvnProjectsDataByProjectName(projectName)
	if len(svnProjectMap) <= 0 {
		return "当前没有svn工程信息，请配置：\n" + GetSvnProjectConfigHelp()
	}

	for _, v := range svnProjectMap {
		if !JudgeIsSearchAllParam(searchValue) && v.ProjectName != searchValue {
			//数据量不大，这里就不再做获取到了退出循环吧
			continue
		}
		result += fmt.Sprintln(tool.MarshalJson(v) + "\n")
	}
	if result == "" {
		return "当前没有符合条件的svn工程配置信息，请配置：\n" + GetSvnProjectConfigHelp()
	} else {
		return "\n***********************以下是svn工程配置数据***********************\n" + result
	}
}

//获取svn工程配置帮助提示
func GetSvnProjectConfigHelp() string {
	tpl := SvnProjectModel{
		ProjectName:              "名称",
		ProjectPath:              "工程的绝对路径",
		ConflictAutoWayWhenMerge: "合并冲突时的自动处理方式：p,mf,tf等",
		AutoBuildMethodList:      []string{},
	}
	return fmt.Sprintf("例：\n【%s：%s】 \n其中路径不能有反斜杠,用/，AutoBuildMethodList对应客户端AutoBuild.cs定义的构建方法数组，如多个配置用分号分割",
		commandName[CommandType_UpdateSvnProjectConfig], tool.MarshalJson(tpl))
}

//获取合并指令帮助
func GetMergeCommandHelp() string {
	return fmt.Sprintf("例：【%s：开发分支合并到策划分支】，开发分支和策划分支都是指令【%s】的ProjectName\n具体分支关系参见https://www.kdocs.cn/l/spWN1ZyWsEPr?f=131",
		commandName[CommandType_SvnMerge],commandName[CommandType_UpdateSvnProjectConfig])
}

//获取客户端构建帮助
func GetClientBuildCommandHelp() string {
	return fmt.Sprintf("例：【%s：外网测试包,BuildLuaCode】或【%s：外网测试包,0】\n参数1和2是指令【%s】里的ProjectName和AutoBuildMethodList方法数组中某个构建方法或其索引\n参数3选填，目前只有固定dev表示是development build，不填则表示默认的release build",
		commandName[CommandType_AutoBuildClient], commandName[CommandType_AutoBuildClient],commandName[CommandType_UpdateSvnProjectConfig])
}

//判断工程是否存在
func JudgeSvnProjectIsExist(projectName, svnProjectName string) bool {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
	projectModel := getSvnProjectData(projectName, svnProjectName)
	return nil != projectModel
}

//获取svn地址
func GetSvnPath(projectName, svnProjectName string) string {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
	svnProjectModel := getSvnProjectData(projectName, svnProjectName)
	if nil == svnProjectModel {
		log.Error("获取工程svn地址，不存在svn工程，请添加")
		return ""
	}
	return svnProjectModel.SvnUrl
}

//获取svn工程地址
func GetSvnProjectPath(projectName, svnProjectName string) string {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
	svnProjectModel := getSvnProjectData(projectName, svnProjectName)
	if nil == svnProjectModel {
		log.Error("获取工程地址，不存在svn工程数据，请添加")
		return ""
	}
	return svnProjectModel.ProjectPath
}

//获取冲突自动处理方式
func GetConflictAutoWayWhenMerge(projectName, svnProjectName string) string {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
	svnProjectModel := getSvnProjectData(projectName, svnProjectName)
	if nil == svnProjectModel {
		log.Error("获取冲突自动处理方式失败，不存在svn工程数据，请添加")
		return ""
	}
	return svnProjectModel.ConflictAutoWayWhenMerge
}

//根据参数获取构建方法
func GetBuildMethod(projectName, svnProjectName, buildMethodParam string) (error, string) {
	intParam, err2Int := strconv.Atoi(buildMethodParam)
	if err2Int != nil {
		intParam = -1
	}
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
	svnProjectModel := getSvnProjectData(projectName, svnProjectName)
	if nil == svnProjectModel {
		return errors.New("获取构建方法失败，不存在svn工程数据，请添加"), ""
	}
	for k, v := range svnProjectModel.AutoBuildMethodList {
		if k == intParam || v == buildMethodParam {
			return nil, v
		}
	}
	return errors.New("获取构建方法失败：" + buildMethodParam), ""
}

//获取上次获取svn日志时间
func GetSvnLogTime(projectName, svnProjectName string) int64 {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
	svnProjectModel := getSvnProjectData(projectName, svnProjectName)
	if nil == svnProjectModel {
		log.Error("获取上次获取svn日志时间，不存在工程，请添加")
		return 0
	}
	return svnProjectModel.LastGetSvnLogTime
}

//保存获取svn日志时间
func SaveSvnLogTime(projectName, svnProjectName string, getLogTime int64) {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
	var fileName string
	fileName, svnProjectMap = getSvnProjectsDataByProjectName(projectName)
	if _, ok := svnProjectMap[svnProjectName]; ok {
		svnProjectMap[svnProjectName].LastGetSvnLogTime = getLogTime
		tool.SaveGobFile(fileName, svnProjectMap)
	} else {
		log.Error("保存svn获取时间失败，不存在svn工程")
	}
}

//获取工程配置
func getSvnProjectData(projectName, svnProjectName string) *SvnProjectModel {
	if projectName == "" || svnProjectName == "" {
		return nil
	}
	_, svnProjectMap = getSvnProjectsDataByProjectName(projectName)
	if _svnProjectModel, ok := svnProjectMap[svnProjectName]; ok {
		return _svnProjectModel
	} else {
		log.Error(svnProjectName, "svn工程配置不存在，请添加")
		return nil
	}
}

//根据webHook获取该项目svn工程数据文件名和数据
func getSvnProjectsDataByProjectName(projectName string) (string, map[string]*SvnProjectModel) {
	svnProjectDataFileName := "svnProject.gob"
	fileName := ProjectName2Md5(projectName) + svnProjectDataFileName
	if fileName == lastProjectFileName {
		return fileName, svnProjectMap
	}
	svnProjectMap = make(map[string]*SvnProjectModel)
	tool.ReadGobFile(fileName, &svnProjectMap)
	lastProjectFileName = fileName
	return fileName, svnProjectMap
}
