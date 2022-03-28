package models

import (
	"autobuildrobot/log"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

//指令类型
const (
	CommandType_Help                    = 0  //帮助
	CommandType_UpdateProjectConfig     = 1  //更新项目配置
	CommandType_UpdateSvnProjectConfig  = 2  //更新svn工程配置
	CommandType_UpdateCdnConfig         = 3  //更新cdn配置
	CommandType_SvnMerge                = 4  //分支合并
	CommandType_UpdateTable             = 5  //更新表格
	CommandType_AutoBuildClient         = 6  //客户端自动构建
	CommandType_PrintHotfixResList      = 7  //输出热更资源列表
	CommandType_UploadHotfixRes2Test    = 8  //上传测试热更资源
	CommandType_UploadHotfixRes2Release = 9  //上传正式热更资源
	CommandType_BackupHotfixRes         = 10 //备份热更资源
	CommandType_UpdateSvrProgressConfig = 11 //更新游戏服务进程配置
	CommandType_UpdateSvrMachineConfig  = 12 //更新游戏服主机配置
	CommandType_UpdateAndRestartSvr     = 13 //更新并重启外网测试服
	CommandType_ListSvnLog              = 14 //打印svn日志
	CommandType_UpdateUser              = 15 //更新用户
	CommandType_CloseRobot              = 16 //关闭机器人
	CommandType_Max                     = 17
)

//自动构建指令
type AutoBuildCommand struct {
	CommandType   int                  //指令类型
	Command       string               //指令
	Name          string               //指令名字
	CommandParams string               //指令参数
	HelpTips      string               //帮助提示
	Func          autoBuildCommandFunc //指令处理函数
	ProjectName   string               //项目名称（如钉钉群标题，一个群一个项目）
	WebHook       string               //回调地址
	ResultFunc    AutoBuildResultFunc  //结果处理函数
}

type autoBuildCommandFunc func(autoBuildCommand AutoBuildCommand) (string, error) //指令处理函数指针
type AutoBuildResultFunc func(msg, executorPhoneNum string)                       //自动构建结果处理函数
var autoBuildCommandMap map[int]AutoBuildCommand
var command [CommandType_Max]string         //指令
var commandName [CommandType_Max]string     //指令名字
var commandHelpTips [CommandType_Max]string //指令帮助提示
var autoBuildCommandRWLock sync.RWMutex

func init() {
	autoBuildCommandMap = make(map[int]AutoBuildCommand)

	//初始化指令(shell脚本文件名，如果不用shell的则不用赋值)
	command[CommandType_SvnMerge] = "svnmerge.sh"
	command[CommandType_AutoBuildClient] = "autoBuildClient.sh"
	command[CommandType_UpdateTable] = "ReadExcel.sh"
	command[CommandType_UpdateAndRestartSvr] = "auto_update_server.py"
	command[CommandType_CloseRobot] = "stopAutoBuildRobot.sh"

	//初始化指令名字
	commandName[CommandType_Help] = "帮助"
	commandName[CommandType_UpdateProjectConfig] = "配置项目"
	commandName[CommandType_UpdateSvnProjectConfig] = "配置svn工程"
	commandName[CommandType_UpdateCdnConfig] = "配置cdn"
	commandName[CommandType_SvnMerge] = "分支合并"
	commandName[CommandType_AutoBuildClient] = "客户端构建"
	commandName[CommandType_PrintHotfixResList] = "输出热更资源"
	commandName[CommandType_UploadHotfixRes2Test] = "上传热更资源到测试"
	commandName[CommandType_UploadHotfixRes2Release] = "上传热更资源到正式"
	commandName[CommandType_BackupHotfixRes] = "备份热更资源"
	commandName[CommandType_UpdateSvrProgressConfig] = "配置游戏服务"
	commandName[CommandType_UpdateSvrMachineConfig] = "配置服务器主机"
	commandName[CommandType_UpdateAndRestartSvr] = "更新服务器"
	commandName[CommandType_ListSvnLog] = "输出svn日志"
	commandName[CommandType_UpdateUser] = "更新用户"
	commandName[CommandType_UpdateTable] = "更新表格"
	commandName[CommandType_CloseRobot] = "关闭自动构建机器人"

	//初始化指令帮助提示
	commandHelpTips[CommandType_UpdateProjectConfig] = GetProjectConfigHelp()
	commandHelpTips[CommandType_UpdateSvnProjectConfig] = GetSvnProjectConfigHelp()
	commandHelpTips[CommandType_UpdateCdnConfig] = GetCdnConfigHelp()
	commandHelpTips[CommandType_SvnMerge] = GetMergeCommandHelp()
	commandHelpTips[CommandType_AutoBuildClient] = GetClientBuildCommandHelp()
	commonHelpTips := "例：【%s：%s】，参数（%s）是svn工程配置的ProjectName（指令【更新svn工程配置】不带参数可以列出所有svn工程配置）"
	commandHelpTips[CommandType_PrintHotfixResList] = fmt.Sprintf(commonHelpTips, commandName[CommandType_PrintHotfixResList], "外网测试包", "外网测试包")
	commandHelpTips[CommandType_UploadHotfixRes2Test] = fmt.Sprintf(commonHelpTips, commandName[CommandType_UploadHotfixRes2Test], "外网测试包", "外网测试包")
	commandHelpTips[CommandType_UploadHotfixRes2Release] = fmt.Sprintf(commonHelpTips, commandName[CommandType_UploadHotfixRes2Release], "外网测试包", "外网测试包")
	commandHelpTips[CommandType_BackupHotfixRes] = fmt.Sprintf("例：【%s：外网测试包,热更日志】，参数1（外网测试包）是svn工程配置的ProjectName（指令【更新svn工程配置】不带参数可以列出所有svn工程配置）\n参数2是备份日志", commandName[CommandType_BackupHotfixRes])
	commandHelpTips[CommandType_UpdateSvrProgressConfig] = GetSvrProgressConfigHelp()
	commandHelpTips[CommandType_UpdateSvrMachineConfig] = GetSvrMachineConfigHelp()
	commandHelpTips[CommandType_UpdateAndRestartSvr] = fmt.Sprintf("例：【%s：外网测试服,后台】,其中外网测试服是%s的配置数据的服务器主机名，后台是%s的配置数据的游戏服务进程名",
		commandName[CommandType_UpdateAndRestartSvr], commandName[CommandType_UpdateSvrMachineConfig], commandName[CommandType_UpdateSvrProgressConfig])
	commandHelpTips[CommandType_ListSvnLog] = fmt.Sprintf(commonHelpTips, commandName[CommandType_ListSvnLog], "开发分支", "开发分支")
	commandHelpTips[CommandType_UpdateUser] = GetUserConfigHelp()
	commandHelpTips[CommandType_UpdateTable] = fmt.Sprintf(commonHelpTips, commandName[CommandType_UpdateTable], "研发表格", "研发表格")
	commandHelpTips[CommandType_CloseRobot] = ""
}

//添加指令
func AddCommand(commandType int, commandFunc autoBuildCommandFunc) {
	if commandType < CommandType_Help || commandType >= CommandType_Max {
		log.Error(fmt.Sprintf("添加越界指令，指令范围：%d-%d", CommandType_Help, CommandType_Max))
		return
	}
	if _, ok := autoBuildCommandMap[commandType]; ok {
		log.Error(fmt.Sprintf("添加重复指令：%d,请检查", commandType))
		return
	}
	autoBuildCommand := AutoBuildCommand{}
	autoBuildCommand.CommandType = commandType
	autoBuildCommand.Command = command[commandType]
	autoBuildCommand.Name = commandName[commandType]
	autoBuildCommand.HelpTips = commandHelpTips[commandType]
	autoBuildCommand.Func = commandFunc
	autoBuildCommandMap[commandType] = autoBuildCommand
}

//获取指令
func GetCommand(commandType int) (autoBuildCommand AutoBuildCommand, ok bool) {
	autoBuildCommandRWLock.RLock()
	defer autoBuildCommandRWLock.RUnlock()
	autoBuildCommand, ok = autoBuildCommandMap[commandType]
	return
}

//获取指令帮助信息
func GetCommandHelpInfo() (help string) {
	autoBuildCommandRWLock.RLock()
	defer autoBuildCommandRWLock.RUnlock()
	for i := 0; i < CommandType_Max; i++ {
		command, ok := autoBuildCommandMap[i]
		if !ok {
			errs := fmt.Sprintf("不存在编号为%d的指令，请添加！", i)
			help += errs
			log.Error(errs)
			continue
		}
		help += fmt.Sprintf("%d:%s\n", i, command.Name)
	}
	return
}

//解析指令
func AnalysisCommand(rawCommand string) (ok bool, autoBuildCommand AutoBuildCommand) {
	//解析指令,先分割参数
	paramSeparators := []string{":", "："}
	requestCommand := rawCommand
	requestParam := ""
	separatorIndex := 99999
	for _, v := range paramSeparators {
		//找到第一个包含分隔符的，通过索引比较，避免分割了带分隔符的参数
		tempIndex := strings.Index(rawCommand, v)
		if tempIndex >= 0 && tempIndex < separatorIndex {
			//有参数
			separatorIndex = tempIndex
			commands := strings.SplitN(rawCommand, v, 2)

			//参数去掉空格和换行
			requestCommand = commands[0]
			requestParam = strings.TrimSpace(commands[1])
			requestParam = strings.Replace(requestParam, "\n", "", -1)
		}
	}

	//获取指令信息
	autoBuildCommandRWLock.RLock()
	requestCommand = strings.Replace(requestCommand, " ", "", -1)
	for _, command := range autoBuildCommandMap {
		if strings.Compare(requestCommand, strconv.Itoa(command.CommandType)) == 0 ||
			strings.Compare(requestCommand, command.Name) == 0 {
			autoBuildCommand = command
			ok = true
			break
		}
	}

	//获取指令信息
	if !ok {
		autoBuildCommand = autoBuildCommandMap[CommandType_Help]
	}
	autoBuildCommand.CommandParams = requestParam
	autoBuildCommandRWLock.RUnlock()
	return
}

//获取指令名字
func GetCommandNameByType(commandType int) string {
	if commandType < CommandType_Help || commandType >= CommandType_Max {
		return "不存在指令类型：" + strconv.Itoa(commandType)
	}
	return commandName[commandType]
}

//判断指令参数是否帮助
func JudgeIsHelpParam(commandParams string) bool {
	return commandParams == "帮助" || strings.ToLower(commandParams) == "help"
}

//判断是否需要项目权限
func JudgeIsNeedProjectPermission(commandType int) bool {
	if commandType == CommandType_SvnMerge || commandType == CommandType_UpdateTable ||
		commandType == CommandType_AutoBuildClient || commandType == CommandType_PrintHotfixResList ||
		commandType == CommandType_UploadHotfixRes2Test || commandType == CommandType_UploadHotfixRes2Release ||
		commandType == CommandType_BackupHotfixRes || commandType == CommandType_UpdateAndRestartSvr {
		return true
	}
	return false
}

//解析参数
func AnalysisParam(requestParam string, commandType int) (err error, params []string) {
	//不需要参数
	if commandType == CommandType_Help || commandType == CommandType_CloseRobot {
		return
	}

	//参数不足
	if requestParam == "" {
		return errors.New("参数不足，请输入帮助获取提示！！！"), nil
	}

	if commandType == CommandType_SvnMerge {
		//特殊处理项目合并，需要两个参数
		for _, flag := range mergeFlags {
			branches := strings.Split(requestParam, flag)
			if len(branches) >= 2 && branches[0] != "" && branches[1] != "" {
				return nil, branches
			}
		}
		return errors.New("获取合并分支失败！"), nil
	} else {
		//根据逗号分割参数
		requestParams := strings.Split(requestParam, ",")
		return nil, requestParams
	}
}

//获取shell指令参数
func GetShellParams(commandType int, commandParams []string, projectName, webHook string) (error, string) {
	//不需要参数
	if commandType == CommandType_Help || commandType == CommandType_CloseRobot {
		return nil,""
	}

	//先判断是否是服务器更新
	if commandType == CommandType_UpdateAndRestartSvr {
		if len(commandParams) <= 1 {
			return errors.New("参数不足"), ""
		}

		//依次为projectPath svrProgressProjDirName platform zipFileName zipDirList zipFileList upload_ip port account psd uploadPath updateCommand
		err, ip, port, account, psd, platform, svrRootPath, projectPath := GetSvrMachineData(projectName, commandParams[0])
		if err != nil {
			return err, ""
		}
		err, dirName, zipFileNameWithoutExt, zipFileList, zipDirList := GetSvrProgressData(projectName, commandParams[1])
		if err != nil {
			return err, ""
		}
		return nil, fmt.Sprintf("\"%s\" \"%s\" \"%s\" \"%s\" \"%s\"  \"%s\"  \"%s\"  \"%s\"  \"%s\"  \"%s\"  \"%s\"",
			projectPath, dirName, platform, zipFileNameWithoutExt, zipDirList, zipFileList, ip, port, account, psd, svrRootPath)
	}

	if len(commandParams) <= 0 {
		return errors.New("参数不足"), ""
	}
	svnProjectName1 := commandParams[0]
	if svnProjectName1 == "" {
		return errors.New(fmt.Sprintf("需要项目名称参数，，用【%s】空参数会列出所有项目配置", GetCommandNameByType(CommandType_UpdateSvnProjectConfig))), ""
	}

	projectPath := GetSvnProjectPath(projectName, svnProjectName1)

	switch commandType {
	case CommandType_SvnMerge:
		{
			svnProjectName2 := commandParams[1]
			if svnProjectName1 == "" || svnProjectName2 == "" {
				return errors.New(fmt.Sprintf("合并分支名称不合法（请输入两个正确分支信息如开发分支合并到策划分支），branch1：%s,branch2:%s", svnProjectName1, svnProjectName2)), ""
			}
			svnUrl := GetSvnPath(projectName, svnProjectName1)
			if "" == svnUrl {
				return errors.New("合并工程svn地址为空：" + svnProjectName1), ""
			}
			mergeTargetProjectPath := GetSvnProjectPath(projectName, svnProjectName2)
			if "" == mergeTargetProjectPath {
				return errors.New("合并目标工程地址为空：" + svnProjectName2), ""
			}
			conflictAutoWayWhenMerge := GetConflictAutoWayWhenMerge(projectName, svnProjectName2)
			//参数依次为合并目标工程地址、合并svn地址、冲突解决方式、合并日志
			return nil, fmt.Sprintf("\"%s\" \"%s\" %s %s", mergeTargetProjectPath,
				svnUrl, conflictAutoWayWhenMerge, fmt.Sprintf("%s合并到%s", svnProjectName1, svnProjectName2))
		}
	case CommandType_UpdateTable:
		{
			return nil, fmt.Sprintf("\"%s\" %s", projectPath, runtime.GOOS)
		}
	case CommandType_AutoBuildClient:
		{
			if svnProjectName1 == "" {
				return errors.New(fmt.Sprintf("需要项目名称参数，，用【%s】空参数会列出所有项目配置", GetCommandNameByType(CommandType_UpdateSvnProjectConfig))), ""
			}
			err, enginePath := GetProjectClientEnginePath(projectName)
			if nil != err {
				return err, ""
			}

			//获取构建方法
			if len(commandParams) < 2 {
				return errors.New(fmt.Sprintf("获取构建方法失败，，用【%s】空参数查看项目配置是否存在该方法", GetCommandNameByType(CommandType_UpdateSvnProjectConfig))), ""
			}
			err, buildMethod := GetBuildMethod(projectName, svnProjectName1, commandParams[1])
			if nil != err {
				return err, ""
			}
			if len(commandParams) <= 2 {
				//依次需要客户端引擎路径、工程路径、构建方法、webhook
				return nil, fmt.Sprintf("\"%s\" \"%s\" \"%s\" \"%s\"", enginePath, projectPath, buildMethod, webHook)
			}
			//默认两个参数分别为项目名和构建方法，如果有多余两个参数则统一作为额外参数
			return nil, fmt.Sprintf("\"%s\" \"%s\" \"%s\" \"%s\" \"%s\"", enginePath, projectPath, buildMethod, webHook, commandParams[2])
		}
	}
	return nil, ""
}
