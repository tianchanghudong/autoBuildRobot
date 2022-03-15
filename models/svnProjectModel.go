package models

import (
	"autobuildrobot/log"
	"autobuildrobot/tool"
	"errors"
	"fmt"
	"runtime"
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
var mergeFlag = []string{"合并到", "合并"}         //项目合并标识，按顺序分割获取两个分支
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
func GetAllSvnProjectsDataByProject(projectName string) string {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()

	_, svnProjectMap = getSvnProjectsDataByProjectName(projectName)
	if len(svnProjectMap) <= 0 {
		return "当前没有svn工程信息，请配置：\n" + GetSvnProjectConfigHelp()
	}

	result := "\n***********************以下是已有的svn工程配置数据***********************\n"
	for _, v := range svnProjectMap {
		result += fmt.Sprintln(tool.MarshalJson(v) + "\n")
	}
	return result
}

//获取svn工程配置帮助提示
func GetSvnProjectConfigHelp() string {
	tpl := SvnProjectModel{
		ProjectName:              "svn工程名称",
		ProjectPath:              "工程的绝对路径,注意不能有反斜杠,用/",
		ConflictAutoWayWhenMerge: "合并冲突时的自动处理方式：p,mf,tf等",
		AutoBuildMethodList:      []string{"客户端自动构建方法名，如打lua代码BuildLuaCode", "打安卓白包方法BuildAndroidApk_Bai，后面依次增加"},
	}
	return fmt.Sprintf("例：\n【%s：%s】 \n多个配置用英文分号分割", commandName[CommandType_UpdateSvnProjectConfig], tool.MarshalJson(tpl))
}

//获取合并指令帮助
func GetMergeCommandHelp() string {
	return fmt.Sprintf("例：【%s：开发分支合并到策划分支】，开发分支和策划分支都是svn工程配置的ProjectName（指令【更新svn工程配置】不带参数可以列出所有svn工程配置）\n具体分支关系参见https://www.kdocs.cn/l/spWN1ZyWsEPr?f=131", commandName[CommandType_SvnMerge])
}

//获取客户端构建帮助
func GetClientBuildCommandHelp() string {
	return fmt.Sprintf("例：【%s：外网测试包,BuildLuaCode】或【%s：外网测试包,0】，\n其中前两个参数是svn工程配置内容（指令【更新svn工程配置】不带参数可以列出所有svn工程配置）\n参数1是svn工程配置的ProjectName\n参数2是svn工程配置的AutoBuildMethodList方法数组中某个构建方法或其索引\n参数3选填，目前只有固定dev表示是development build，不填则表示默认的release build", commandName[CommandType_AutoBuildClient], commandName[CommandType_AutoBuildClient])
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

//提取指令参数里的svn工程名称
func GetSvnProjectName(requestParam string, commandType int) (project1, project2 string, err error) {
	//先判断操作是否需要工程名称
	if commandType != CommandType_SvnMerge && commandType != CommandType_AutoBuildClient &&
		commandType != CommandType_PrintHotfixResList && commandType != CommandType_BackupHotfixRes &&
		commandType != CommandType_UploadHotfixRes2Test && commandType != CommandType_UploadHotfixRes2Release &&
		commandType != CommandType_UpdateAndRestartIntranetServer && commandType != CommandType_UpdateAndRestartExtranetTestServer &&
		commandType != CommandType_ListSvnLog && commandType != CommandType_UpdateTable {
		return "", "", nil
	}

	//需要名称但参数不足
	if requestParam == "" {
		return "", "", errors.New("需要至少有svn工程名称参数！！！")
	}

	//这些固定第一个参数是工程名称
	requestParams := strings.Split(requestParam, ",")
	if len(requestParams) <= 0 {
		return "", "", errors.New("获取svn工程名称失败")
	}

	if commandType == CommandType_SvnMerge {
		//特殊处理项目合并，需要两个分支
		for _, flag := range mergeFlag {
			branches := strings.Split(requestParams[0], flag)
			if len(branches) >= 2 && branches[0] != "" && branches[1] != ""{
				return branches[0], branches[1], nil
			}
		}
		return "", "", errors.New("获取合并分支失败！")
	}
	return requestParams[0], "", nil
}

//获取shell指令参数
func GetProjectShellParams(projectName, svnProjectName1, svnProjectName2, commandParams, webHook string, commandType int) (error, string) {
	svnProjectDataLock.Lock()
	defer svnProjectDataLock.Unlock()
	svnProjectModel := getSvnProjectData(projectName, svnProjectName1)
	if svnProjectName1 != "" && nil == svnProjectModel {
		return errors.New("不存在svn工程配置：" + svnProjectName1), ""
	}

	switch commandType {
	case CommandType_SvnMerge:
		{
			if svnProjectName1 == "" || svnProjectName2 == "" {
				return errors.New(fmt.Sprintf("合并分支名称不合法（请输入两个正确分支信息如开发分支合并到策划分支），branch1：%s,branch2:%s", svnProjectName1, svnProjectName2)), ""
			}
			mergeTargetBranch := getSvnProjectData(projectName, svnProjectName2)
			if nil == mergeTargetBranch {
				return errors.New("不存在合并目标分支配置：" + svnProjectName2), ""
			}
			//参数依次为合并目标工程地址、合并svn地址、冲突解决方式、合并日志
			return nil, fmt.Sprintf("\"%s\" \"%s\" %s %s", mergeTargetBranch.ProjectPath,
				svnProjectModel.SvnUrl, mergeTargetBranch.ConflictAutoWayWhenMerge, fmt.Sprintf("%s合并到%s", svnProjectModel.ProjectName, mergeTargetBranch.ProjectName))
		}
	case CommandType_UpdateTable:
		{
			if svnProjectName1 == "" {
				return errors.New(fmt.Sprintf("需要项目名称参数，，用【%s】空参数会列出所有项目配置", GetCommandNameByType(CommandType_UpdateSvnProjectConfig))), ""
			}
			return nil, fmt.Sprintf("\"%s\" %s", svnProjectModel.ProjectPath, runtime.GOOS)
		}
	case CommandType_AutoBuildClient:
		{
			if svnProjectName1 == "" {
				return errors.New(fmt.Sprintf("需要项目名称参数，，用【%s】空参数会列出所有项目配置", GetCommandNameByType(CommandType_UpdateSvnProjectConfig))), ""
			}
			enginePath := GetProjectClientEnginePath(projectName)

			//获取构建方法
			err, buildMethod := svnProjectModel.getBuildMethod(commandParams)
			if nil != err {
				return err, ""
			}
			if buildMethod == "" {
				return errors.New(fmt.Sprintf("获取构建方法失败，，用【%s】空参数查看项目配置是否存在该方法", GetCommandNameByType(CommandType_UpdateSvnProjectConfig))), ""
			}
			paramDivisionFlagCount := strings.Count(commandParams, ",")
			if paramDivisionFlagCount <= 1 {
				//依次需要客户端引擎路径、工程路径、构建方法、webhook
				return nil, fmt.Sprintf("\"%s\" \"%s\" \"%s\" \"%s\"", enginePath, svnProjectModel.ProjectPath, buildMethod, webHook)
			}
			//默认两个参数分别为项目名和构建方法，如果有多余两个参数则统一作为额外参数
			params := strings.SplitN(commandParams, ",", 3)
			return nil, fmt.Sprintf("\"%s\" \"%s\" \"%s\" \"%s\" \"%s\"", enginePath, svnProjectModel.ProjectPath, buildMethod, webHook, params[2])
		}
	}
	return nil, commandParams
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

//根据参数获取构建方法
func (this *SvnProjectModel) getBuildMethod(commandParams string) (error, string) {
	requestParams := strings.Split(commandParams, ",")
	if len(requestParams) < 2 {
		return errors.New(fmt.Sprintf("参数不足，【%s：help】获取帮助", GetCommandNameByType(CommandType_AutoBuildClient))), ""
	}
	intParam, err2Int := strconv.Atoi(requestParams[1])
	if err2Int != nil {
		intParam = -1
	}
	buildMethod := ""
	for k, v := range this.AutoBuildMethodList {
		if k == intParam || v == requestParams[1] {
			buildMethod = v
			break
		}
	}
	return nil, buildMethod
}
