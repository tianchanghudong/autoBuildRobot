package models

import (
	"autobuildrobot/log"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

//指令类型
const (
	CommandType_Help                               = 0  //帮助
	CommandType_UpdateProjectConfig                = 1  //更新项目配置
	CommandType_UpdateSvnProjectConfig             = 2  //更新svn工程配置
	CommandType_UpdateCdnConfig                    = 3  //更新cdn配置
	CommandType_SvnMerge                           = 4  //分支合并
	CommandType_UpdateTable                        = 5  //更新表格
	CommandType_AutoBuildClient                    = 6  //客户端自动构建
	CommandType_PrintHotfixResList                 = 7  //输出热更资源列表
	CommandType_UploadHotfixRes2Test               = 8  //上传测试热更资源
	CommandType_UploadHotfixRes2Release            = 9 //上传正式热更资源
	CommandType_BackupHotfixRes                    = 10 //备份热更资源
	CommandType_UpdateAndRestartIntranetServer     = 11 //更新并重启内网服务器
	CommandType_UpdateAndRestartExtranetTestServer = 12 //更新并重启外网测试服
	CommandType_ListSvnLog                         = 13 //打印svn日志
	CommandType_UpdateUser                         = 14 //更新用户
	CommandType_CloseRobot                         = 15 //关闭机器人
	CommandType_ExcuteSeriesCommand                = 16 //执行多条指令
	CommandType_Max                                = 17
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
	WebHook       string  			   //回调地址
	ResultFunc    AutoBuildResultFunc  //结果处理函数
}

type autoBuildCommandFunc func(autoBuildCommand AutoBuildCommand) string //指令处理函数指针
type AutoBuildResultFunc func(msg, executorPhoneNum string)              //自动构建结果处理函数
var autoBuildCommandMap map[int]AutoBuildCommand
var command [CommandType_Max]string         //指令
var commandName [CommandType_Max]string     //指令名字
var commandHelpTips [CommandType_Max]string //指令帮助提示
var autoBuildCommandRWLock sync.RWMutex

func init() {
	autoBuildCommandMap = make(map[int]AutoBuildCommand)

	//初始化指令(shell脚本文件名，如果不用shell的则不用赋值)
	command[CommandType_SvnMerge] = "svnmerge"
	command[CommandType_AutoBuildClient] = "autoBuildClient"
	command[CommandType_UpdateTable] = "ReadExcel"
	command[CommandType_CloseRobot] = "stopAutoBuildRobot"

	//初始化指令名字
	commandName[CommandType_Help] = "帮助"
	commandName[CommandType_UpdateProjectConfig] = "更新项目配置"
	commandName[CommandType_UpdateSvnProjectConfig] = "更新svn工程配置"
	commandName[CommandType_UpdateCdnConfig] = "更新cdn配置"
	commandName[CommandType_SvnMerge] = "分支合并"
	commandName[CommandType_AutoBuildClient] = "客户端构建"
	commandName[CommandType_PrintHotfixResList] = "输出热更资源"
	commandName[CommandType_UploadHotfixRes2Test] = "上传热更资源到测试"
	commandName[CommandType_UploadHotfixRes2Release] = "上传热更资源到正式"
	commandName[CommandType_BackupHotfixRes] = "备份热更资源"
	commandName[CommandType_UpdateAndRestartIntranetServer] = "更新内网服务器"
	commandName[CommandType_UpdateAndRestartExtranetTestServer] = "更新外网测试服"
	commandName[CommandType_ListSvnLog] = "输出svn日志"
	commandName[CommandType_UpdateUser] = "更新用户"
	commandName[CommandType_UpdateTable] = "更新表格"
	commandName[CommandType_CloseRobot] = "关闭自动构建机器人"
	commandName[CommandType_ExcuteSeriesCommand] = "执行多条指令"

	//初始化指令帮助提示
	commandHelpTips[CommandType_UpdateProjectConfig] = GetProjectConfigHelp()
	commandHelpTips[CommandType_UpdateSvnProjectConfig] = GetSvnProjectConfigHelp()
	commandHelpTips[CommandType_UpdateCdnConfig] = GetCdnConfigHelp()
	commandHelpTips[CommandType_SvnMerge] = fmt.Sprintf("例：【%s：开发分支合并到策划分支】，开发分支和策划分支都是svn工程配置的工程名，具体分支关系参见https://www.kdocs.cn/l/spWN1ZyWsEPr?f=131",commandName[CommandType_SvnMerge])
	commandHelpTips[CommandType_AutoBuildClient] = fmt.Sprintf("例：【%s：快接安卓,BuildLuaCode】或【%s：快接安卓,0】，其中参数1是svn工程配置的工程名,参数2是svn工程配置的构建方法或方法索引",commandName[CommandType_AutoBuildClient],commandName[CommandType_AutoBuildClient])
	commandHelpTips[CommandType_PrintHotfixResList] = fmt.Sprintf("例：【%s：快接安卓】，其中快接安卓是svn工程配置的工程名",commandName[CommandType_PrintHotfixResList])
	commandHelpTips[CommandType_UploadHotfixRes2Test] = fmt.Sprintf("例：【%s：快接安卓】，其中快接安卓是svn工程配置的工程名",commandName[CommandType_UploadHotfixRes2Test])
	commandHelpTips[CommandType_UploadHotfixRes2Release] = fmt.Sprintf("例：【%s：快接安卓】，其中快接安卓是svn工程配置的工程名",commandName[CommandType_UploadHotfixRes2Release])
	commandHelpTips[CommandType_BackupHotfixRes] = fmt.Sprintf("例：【%s：快接安卓,热更日志】，其中参数1是svn工程配置的工程名，参数2是备份日志",commandName[CommandType_BackupHotfixRes])
	commandHelpTips[CommandType_UpdateAndRestartIntranetServer] = fmt.Sprintf("例：【%s：内网分支】，其中内网分支是svn工程配置的工程名",commandName[CommandType_UpdateAndRestartIntranetServer])
	commandHelpTips[CommandType_UpdateAndRestartExtranetTestServer] = fmt.Sprintf("例：【%s：外网分支】，其中内网分支是svn工程配置的工程名",commandName[CommandType_UpdateAndRestartExtranetTestServer])
	commandHelpTips[CommandType_ListSvnLog] = fmt.Sprintf("例：【%s：开发分支】，其中开发分支是svn工程配置的工程名",commandName[CommandType_ListSvnLog])
	commandHelpTips[CommandType_UpdateUser] = GetUserConfigHelp()
	commandHelpTips[CommandType_UpdateTable] = fmt.Sprintf("例：【%s：研发表格】，其中研发表格是svn工程配置的工程名",commandName[CommandType_UpdateTable])
	commandHelpTips[CommandType_CloseRobot] = ""
	commandHelpTips[CommandType_ExcuteSeriesCommand] = fmt.Sprintf("例：【%s:分支合并：开发分支合并到策划分支->更新表格：研发表格->分支合并：策划分支合并到测试分支】，冒号后为多条指令集合，每条指令用英文箭头->分割",)
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
	if commandType < CommandType_Help || commandType >= CommandType_Max{
		return "不存在指令类型：" + strconv.Itoa(commandType)
	}
	return commandName[commandType]
}

//判断指令参数是否帮助
func JudgeIsHelpParam(commandParams string) bool {
	return commandParams == "帮助" || strings.ToLower(commandParams) == "help"
}
