package manager

import (
	"autobuildrobot/log"
	"autobuildrobot/models"
	"autobuildrobot/tool"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

//app.conf配置信息
var winGitPath = ""        //window git 安装路径，用于执行shell脚本
var lineInOneMes = 200     //一条构建消息的行数
var logFilter = ""         //svn日志过滤字符串
var closeRobotTime = 3580  //定时关闭机器人得时间（从0点算起得秒数）
var svnMachineAddr = ""    //svn机器地址
var svnMachineAccount = "" //svn机器账号
var svnMachinePwd = ""     //svn机器密码

const (
	shellPath                  = "shell"  //shell脚本地址
	maxIntervalTimeBetween2Msg = int64(5) //一条消息间隔最长时间
)

//初始化
func init() {
	//初始化配置
	temp, _ := beego.GetConfig("String", "winGitPath", "")
	winGitPath = temp.(string)
	temp, _ = beego.GetConfig("Int", "lineInOneMes", 80)
	lineInOneMes = temp.(int)
	temp, _ = beego.GetConfig("String", "logFilter", "")
	logFilter = temp.(string)
	temp, _ = beego.GetConfig("Int", "closeRobotTime", 3580)
	closeRobotTime = temp.(int)
	temp, _ = beego.GetConfig("String", "svnMachineAddr", "")
	svnMachineAddr = temp.(string)
	temp, _ = beego.GetConfig("String", "svnMachineAccount", "")
	svnMachineAccount = temp.(string)
	temp, _ = beego.GetConfig("String", "svnMachinePwd", "")
	svnMachinePwd = temp.(string)
	log.Info(fmt.Sprintf("winGitPath:%s,lineInOneMes：%d,logFilter:%s,closeRobotTime:%d",
		winGitPath, lineInOneMes, logFilter, closeRobotTime))

	//初始化指令处理函数
	models.AddCommand(models.CommandType_Help, helpCommand)
	models.AddCommand(models.CommandType_ProjectConfig, updateProjectConfigCommand)
	models.AddCommand(models.CommandType_SvnProjectConfig, updateSvnProjectConfigCommand)
	models.AddCommand(models.CommandType_CdnConfig, updateCdnConfigCommand)
	models.AddCommand(models.CommandType_CheckOutSvnProject, checkOutSvnProject)
	models.AddCommand(models.CommandType_SvnMerge, shellCommand)
	models.AddCommand(models.CommandType_AutoBuildClient, shellCommand)
	models.AddCommand(models.CommandType_PrintHotfixResList, printHotfixResList)
	models.AddCommand(models.CommandType_UploadHotfixRes2Test, uploadHotfixRes2Test)
	models.AddCommand(models.CommandType_UploadHotfixRes2Release, uploadHotfixRes2Release)
	models.AddCommand(models.CommandType_BackupHotfixRes, backupHotfixRes)
	models.AddCommand(models.CommandType_SvrProgressConfig, updateSvrProgressConfigCommand)
	models.AddCommand(models.CommandType_SvrMachineConfig, updateSvrMachineConfigCommand)
	models.AddCommand(models.CommandType_UpdateAndRestartSvr, shellCommand)
	models.AddCommand(models.CommandType_BuildPbMsg, shellCommand)
	models.AddCommand(models.CommandType_CloseSvr, shellCommand)
	models.AddCommand(models.CommandType_UpdateTable, shellCommand)

	models.AddCommand(models.CommandType_UserGroup, updateUserGroupCommand)
	models.AddCommand(models.CommandType_User, updateUserCommand)
	models.AddCommand(models.CommandType_ListSvnLog, listAllSvnLog)
	models.AddCommand(models.CommandType_CloseRobot, shellCommand)
	models.AddCommand(models.CommandType_TemplateCmd, updateCmdTempConfigCommand)
	models.AddCommand(models.CommandType_BackupSvn, backupSvn)

	//直接在指令执行前先判断并获取模板指令
	models.AddCommand(models.CommandType_ExcuteTemplateCmd, nullCommandFunc)

	//定时关闭机器人（mac要定时关机，但是如果没有结束机器人会卡住关机）
	if closeRobotTime < 0 || runtime.GOOS != "darwin" {
		//小于0则表示不定时关闭
		return
	}
	timeNow := time.Now()
	hour := closeRobotTime / 3600
	minute := (closeRobotTime % 3600) / 60
	second := closeRobotTime % 60
	targetTime := time.Date(timeNow.Year(), timeNow.Month(), timeNow.Day(), hour, minute, second, 0, timeNow.Location())
	targetTime = targetTime.AddDate(0, 0, 1)
	interalTime := targetTime.Unix() - timeNow.Unix()
	log.Info(fmt.Sprintf("还有%d秒将关闭机器人", interalTime))
	time.AfterFunc(time.Duration(interalTime)*time.Second, closeAutoBuildByTime)
}

//定时关闭机器人
func closeAutoBuildByTime() {
	autoBuildCommand, ok := models.GetCommand(models.CommandType_CloseRobot)
	if !ok {
		log.Error("不存在关闭机器人指令")
		return
	}
	log.Info("关闭自动构建机器人")
	autoBuildCommand.Func(&autoBuildCommand)
	time.AfterFunc(24*time.Hour, closeAutoBuildByTime)
}

//收到指令
func RecvCommand(projectName, executor, rawCmd, webHook string, sendResultFunc models.SendResultFunc) {
	phoneNum := models.GetUserPhone(projectName, executor)

	//处理异常
	defer func() {
		if r := recover(); r != nil {
			result := fmt.Errorf("程序异常:%v,大概率网络异常，重试一次试试！", r).Error()
			sendResultFunc(fmt.Sprintf("builder:%s\ninfo:%s", executor, result), phoneNum)
		}
	}()

	//解析指令
	errAnalysisCommand, commandList := models.AnalysisCommand(projectName, rawCmd)
	if nil != errAnalysisCommand {
		sendResultFunc(fmt.Sprintf("执行异常：%s", errAnalysisCommand.Error()), phoneNum)
		return
	}

	//执行指令
	cmdCount := len(commandList)
	for cmdIndex, autoBuildCommand := range commandList {
		//先通知构建群操我已经在处理
		result := fmt.Sprintf("开始执行%d/%d指令：【%s:%s】...", cmdIndex+1, cmdCount, autoBuildCommand.Name, autoBuildCommand.CommandParams)
		sendResultFunc(fmt.Sprintf("builder:%s\ninfo:%s", executor, result), "")
		time.Sleep(time.Duration(1) * time.Millisecond)

		//执行指令
		commandResult := ""
		autoBuildCommand.WebHook = webHook
		autoBuildCommand.SendResult = sendResultFunc
		autoBuildCommand.ProjectName = projectName
		if models.JudgeIsHelpParam(autoBuildCommand.CommandParams) {
			//如果参数是帮助，则返回指令帮助信息
			commandResult = autoBuildCommand.HelpTips
		} else {
			//判断是否有权限
			isHavePermission, tips := models.JudgeIsHadPermission(autoBuildCommand.CommandType, projectName, executor, autoBuildCommand.CommandParams)
			if !isHavePermission {
				phoneNum += "," + models.GetProjectManagerPhone(projectName)
				sendResultFunc(fmt.Sprintf("builder:%s\ninfo:%s", executor, tips), phoneNum)
				return
			}

			//返回指令执行结果
			var err error
			commandResult, err = autoBuildCommand.Func(autoBuildCommand)
			if err != nil {
				sendResultFunc(fmt.Sprintf("builder:%s\ninfo:%s", executor, err.Error()), phoneNum)
				return
			}
		}

		//发送执行结果
		sendResultFunc(fmt.Sprintf("builder:%s\ncommand:%s:%s\ninfo:%s", executor, autoBuildCommand.Name, autoBuildCommand.CommandParams, commandResult), phoneNum)
	}

	if cmdCount > 1 {
		//多条指令
		sendResultFunc(fmt.Sprintf("builder:%s\ninfo:所有指令执行完成,【%s】", executor, rawCmd), phoneNum)
	}
}

//执行帮助指令
func helpCommand(command *models.AutoBuildCommand) (string, error) {
	return models.GetCommandHelpInfo(command.ProjectName), nil
}

//执行更新项目配置指令
func updateProjectConfigCommand(command *models.AutoBuildCommand) (string, error) {
	projectConfig := command.CommandParams
	if strings.Contains(projectConfig, "{") {
		//更新
		return models.UpdateProject(command.ProjectName, projectConfig)
	} else {
		//获取项目数据
		return models.GetProjectData(command.ProjectName), nil
	}
}

//执行更新svn工程配置指令
func updateSvnProjectConfigCommand(command *models.AutoBuildCommand) (string, error) {
	svnProjectConfig := command.CommandParams
	if strings.Contains(svnProjectConfig, "{") {
		//更新svn工程数据
		return models.UpdateSvnProject(command.ProjectName, svnProjectConfig), nil
	} else {
		//查询配置数据
		return models.QuerySvnProjectsDataByProject(command.ProjectName, svnProjectConfig), nil
	}
}

//更新cdn配置
func updateCdnConfigCommand(command *models.AutoBuildCommand) (string, error) {
	cdnConfig := command.CommandParams
	if strings.Contains(cdnConfig, "{") {
		//更新cdn配置
		return models.UpdateCdn(command.ProjectName, cdnConfig), nil
	} else {
		//查询cdn配置数据
		return models.QueryCdnDataOfOneProject(command.ProjectName, cdnConfig), nil
	}
}

//更新服务进程配置
func updateSvrProgressConfigCommand(command *models.AutoBuildCommand) (string, error) {
	svrConfig := command.CommandParams
	if strings.Contains(svrConfig, "{") {
		//更新svrProgress配置
		return models.UpdateSvrProgressData(command.ProjectName, svrConfig), nil
	} else {
		//搜索svrProgress配置
		return models.QueryProgressDataOfOneProject(command.ProjectName, svrConfig), nil
	}
}

//更新服务主机配置
func updateSvrMachineConfigCommand(command *models.AutoBuildCommand) (string, error) {
	svrMachineConfig := command.CommandParams
	if strings.Contains(svrMachineConfig, "{") {
		//更新svrMachine配置
		return models.UpdateSvrMachineData(command.ProjectName, svrMachineConfig), nil
	} else {
		//查询数据
		return models.QuerySvrMachineDataOfOneProject(command.ProjectName, svrMachineConfig), nil
	}
}

//检出svn工程
func checkOutSvnProject(command *models.AutoBuildCommand) (string, error) {
	//解析指令
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return "", err
	}

	//获取svn工程配置
	err, projectPath, svnUrl, _ := models.GetSvnProjectInfo(command.ProjectName, params[0])
	if nil != err {
		return "", err
	}

	commandName := "sh"
	if runtime.GOOS == "windows" {
		commandName = winGitPath
	}

	//如果路径存在，则输出svn信息
	isExist, err := tool.PathExists(projectPath)
	if nil != err {
		return "", err
	}
	if isExist {
		svnInfoCommand := fmt.Sprintf("cd %s;svn info", projectPath)
		result, err := tool.Exec_shell(commandName, svnInfoCommand)
		return fmt.Sprintf("已存在【%s】svn工程!!\n%s", params[0], result), err
	}

	//创建文件夹
	dirErr := tool.CreateDir(projectPath)
	if nil != dirErr {
		return "", dirErr
	}

	//检出svn
	count := 0
	temp := ""
	lastTime := time.Now().Unix()
	svnCheckOutCommand := fmt.Sprintf("cd %s;svn co %s .", projectPath, svnUrl)
	tool.ExecCommand(commandName, svnCheckOutCommand, func(resultLine string) {
		timeNow := time.Now().Unix()
		if (timeNow - lastTime) < maxIntervalTimeBetween2Msg {
			//过滤掉一些消息，要不消息太恐怖了
			return
		}

		//每隔80行发送一条构建消息
		count++
		temp += resultLine
		if count >= lineInOneMes {
			command.SendResult(temp, "")
			temp = ""
			count = 0
			lastTime = timeNow
		}
	})
	return temp + "\n检出完毕（避免消息爆炸，有过滤一些svn检出日志）！！", nil
}

//执行shell指令
func shellCommand(command *models.AutoBuildCommand) (string, error) {
	//检测外链是否正确（只有构建客户端和更新服务器会检测）
	if errCheckExternal := checkSVNExternals(command); errCheckExternal != nil {
		return "", errCheckExternal
	}

	//获取指令
	commandTxt := command.Command
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return "", err
	}
	err, shellParams := models.GetShellParams(command.CommandType, params, command.ProjectName, command.WebHook)
	if nil != err {
		return "", err
	}

	if path.Ext(commandTxt) == ".py" {
		commandTxt = fmt.Sprintf("cd %s;chmod +x %s;python %s %s", shellPath, commandTxt, commandTxt, shellParams)
	} else {
		commandTxt = fmt.Sprintf("cd %s;chmod +x %s;./%s %s", shellPath, commandTxt, commandTxt, shellParams)
	}

	if commandTxt == "" {
		return "", errors.New("shellCommand,指令为空，请检查！！！")
	}

	//执行指令
	temp := ""
	count := 0
	commandName := "sh"
	if runtime.GOOS == "windows" {
		commandName = winGitPath
	}
	result := ""
	isError := false
	lastTime := time.Now().Unix()
	tool.ExecCommand(commandName, commandTxt, func(resultLine string) {
		//简单判断是否异常吧
		lowerResult := strings.ToLower(resultLine)
		if !strings.Contains(resultLine, "    ") && (strings.Contains(resultLine, "异常") || strings.Contains(lowerResult, "exception") ||
			strings.Contains(resultLine, "失败") || strings.Contains(lowerResult, "fail")) {
			//svn输出都包含"    "，这里简单通过这个过滤掉svn文件名-_-
			isError = true
		}

		//每隔80行发送一条构建消息
		count++
		temp += resultLine
		timeNow := time.Now().Unix()
		if count >= lineInOneMes || (timeNow-lastTime) > maxIntervalTimeBetween2Msg {
			command.SendResult(temp, "")
			temp = ""
			count = 0
			lastTime = timeNow
		}
	})

	//检测冲突
	if errCheckConflict := checkSVNConflictAndNotifyManager(command); errCheckConflict != nil {
		temp += "\n" + errCheckConflict.Error()
		isError = true
	}

	//如果是导表或者打消息码，则要提交或者还原
	if command.CommandType == models.CommandType_UpdateTable ||
		command.CommandType == models.CommandType_BuildPbMsg {
		_, projectPath, _, _ := models.GetSvnProjectInfo(command.ProjectName, params[0])
		if isError {
			//还原
			revertCommand := fmt.Sprintf("cd %s;svn revert -R .;", projectPath)
			tool.Exec_shell(commandName, revertCommand)
		} else {
			//提交
			commitCommand := fmt.Sprintf("cd %s;chmod +x svnCommit.sh;./svnCommit.sh %s %s", shellPath, projectPath, "latest!!!")
			_result, _ := tool.Exec_shell(commandName, commitCommand)
			temp += "\n" + _result
		}
	}

	if isError {
		result = temp + "\n" + fmt.Sprintf("执行【%s:%s】异常！！！！", command.Name, command.CommandParams)
		return "", errors.New(result)
	} else {
		result = temp + "\n" + fmt.Sprintf("执行【%s:%s】完毕！", command.Name, command.CommandParams)
		return result, nil
	}
}

//检测冲突并且通知管理员
func checkSVNConflictAndNotifyManager(command *models.AutoBuildCommand) error {
	if command.CommandType != models.CommandType_SvnMerge &&
		command.CommandType != models.CommandType_UpdateTable &&
		command.CommandType != models.CommandType_BuildPbMsg {
		return nil
	}

	//检测冲突
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return err
	}
	svnProjectName := params[0]
	err, projectPath, _, _ := models.GetSvnProjectInfo(command.ProjectName, svnProjectName)
	if nil != err {
		return err
	}
	commandName := "sh"
	if runtime.GOOS == "windows" {
		commandName = winGitPath
	}
	checkMergeCommand := fmt.Sprintf("cd %s;svn st -q|grep 'C '", projectPath)
	checkMergeResult, _ := tool.Exec_shell(commandName, checkMergeCommand)
	log.Info("检测冲突完毕：" + checkMergeCommand)
	if checkMergeResult != "" {
		mergeErrorTips := "合并冲突或者其他问题，需要手动处理！！"
		managerPhone := models.GetProjectManagerPhone(command.ProjectName)
		log.Error(command.ProjectName + ",managerPhone:" + managerPhone)
		command.SendResult(fmt.Sprintf("info:%s", mergeErrorTips), managerPhone)
		return errors.New("合并冲突！")
	}
	return nil
}

//检测外链是否修改并且通知管理员，正常来说不检测的，合并的时候有还原外链，但是如果冲突手动处理提交就可能修改外链了
func checkSVNExternals(command *models.AutoBuildCommand) (err error) {
	if command.CommandType != models.CommandType_AutoBuildClient && command.CommandType != models.CommandType_UpdateAndRestartSvr {
		return
	}

	//检测外链
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return err
	}
	svnProjectName := params[0]
	err, projectPath, _, svnExternalKeyword := models.GetSvnProjectInfo(command.ProjectName, svnProjectName)
	if nil != err {
		return err
	}
	if "" == svnExternalKeyword {
		//没有配置则表示不用外链
		return
	}
	if "" == projectPath {
		return errors.New("检测外链，获取项目地址失败，请配置！")
	}

	commandName := "sh"
	if runtime.GOOS == "windows" {
		commandName = winGitPath
	}

	getExternalCmd := fmt.Sprintf(`cd %s;svn st | awk '{if ($1 == "X") {print $2} }' | sed 's/\\/\//g' | awk -F / '{for(i=1;i<NF;i++){if(i==NF-1){print $i "/"}else{printf $i "/"}}}'`, projectPath)
	getExternalResult, _ := tool.Exec_shell(commandName, getExternalCmd)
	checkExternalPathList := strings.Split(getExternalResult, "\n")
	tempMap := make(map[string]bool)
	for _, path := range checkExternalPathList {
		if "" == path || strings.Contains(path, "ServerUnit") {
			continue
		}
		if ok, _ := tempMap[path]; ok {
			continue
		}
		tempMap[path] = true
		checkMergeCommand := fmt.Sprintf("svn pg svn:externals %s/%s", projectPath, path)
		checkSvnExternalsResult, _ := tool.Exec_shell(commandName, checkMergeCommand)
		log.Info(fmt.Sprintf("检测外链：%s,result:%s", checkMergeCommand, checkSvnExternalsResult))
		if checkSvnExternalsResult == "" {
			return errors.New("不存在外链，但是有配置检测，请检查！")
		}
		results := strings.Split(checkSvnExternalsResult, "\n")
		for _, v := range results {
			if v == "" {
				continue
			}
			if !strings.Contains(v, svnExternalKeyword) {
				return errors.New("检测到外链跟配置不一致，请检查！")
			}
		}
	}
	return
}

//输出热更资源列表
func printHotfixResList(command *models.AutoBuildCommand) (string, error) {
	//获取项目信息
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return "", err
	}
	svnProjectName := params[0]
	err, projectPath, _, _ := models.GetSvnProjectInfo(command.ProjectName, svnProjectName)
	if err != nil {
		return "", err
	}

	//再获取cdn配置
	err, cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret, _, resPaths := models.GetCdnData(command.ProjectName, svnProjectName)
	if nil != err {
		return "", err
	}

	//判断本地files.txt是否存在
	localFilesPath := path.Join(projectPath, models.CLIENTLOCALRESPATH, models.CLIENTHOTFIXEDFILENAME)
	if !tool.CheckFileIsExist(localFilesPath) {
		return "", errors.New("项目不存在files.txt文件")
	}

	//获取cdn对象
	cdnErr, cdnClient := GetCdnClient(cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret)
	if nil != cdnErr {
		return "", cdnErr
	}

	//判断cdn服务器上目标files.txt是否存在
	isNotExistUpdateFiles := false
	result := ""
	for _, resPath := range resPaths {
		remoteHotfixedFilesPath := path.Join(resPath, models.CLIENTHOTFIXEDFILENAME)
		isExist, fileExistErr := cdnClient.IsExistFile(remoteHotfixedFilesPath)
		if nil != fileExistErr {
			log.Error(fmt.Sprintf("判断测试files.txt文件是否存在错误，资源路径：%s,错误原因：%s", remoteHotfixedFilesPath, fileExistErr.Error()))
			return "", fileExistErr
		}

		//如果不存在则上传本地files
		if isExist {
			continue
		}
		uploadFileErr := cdnClient.UploadFile(localFilesPath, remoteHotfixedFilesPath)
		if nil == uploadFileErr {
			result += fmt.Sprintf("cdn服务器不存在%s文件,已上传本地文件到cdn服务器\n", remoteHotfixedFilesPath)
		} else {
			return "", errors.New(fmt.Sprintf("cdn服务器不存在%s文件且上传本地文件失败，错误原因：%s\n", remoteHotfixedFilesPath, uploadFileErr.Error()))
		}
		isNotExistUpdateFiles = true
	}
	if isNotExistUpdateFiles {
		//不存在files.txt，都是最新的，没有需要热更的东西
		return result, nil
	}

	//获取本地files.txt文件
	localFile, err := os.Open(localFilesPath)
	if err != nil {
		fmt.Println("读取项目本地file.txt失败 os.Open:", err)
		return "", errors.New("读取项目本地file.txt失败 os.Open:" + err.Error())
	}
	localFileByts, err := ioutil.ReadAll(localFile)
	localFile.Close()
	if err != nil {
		return "", errors.New("读取file.txt失败 ioutil.ReadAll:" + err.Error())
	}

	//获取服务器测试热更files.txt
	remoteTestHotfixedFilesPath := path.Join(resPaths[0], models.CLIENTHOTFIXEDFILENAME)
	testFileErr, testFiles := cdnClient.DownFile(remoteTestHotfixedFilesPath)
	if nil != testFileErr {
		return "", errors.New(fmt.Sprintf("获取测试files.txt文件失败，资源路径：%s,错误原因：%s", remoteTestHotfixedFilesPath, testFileErr.Error()))
	}

	//根据本地和服务器files。txt文件比对获取需要更新的数据
	needUpdateHotfixedFilesDataMap := models.GetNeedUpdateDatas(string(localFileByts), string(testFiles))

	//缓存数据并返回结果给钉钉群
	models.ClientHotFixedDataLock.Lock()
	models.ClientHotFixedFileDataTempMap[svnProjectName] = make([]*models.ClientHotFixedFileData, 0)
	temp := "需要更新的文件有：\n"
	count := 0
	totalSize := 0
	for k, v := range needUpdateHotfixedFilesDataMap {
		models.ClientHotFixedFileDataTempMap[svnProjectName] = append(models.ClientHotFixedFileDataTempMap[svnProjectName], v)
		count++
		fileSize, _ := strconv.Atoi(v.Size)
		totalSize += fileSize
		temp += fmt.Sprintf("%s≈%dKB\n", k, (fileSize / 1024))

		//避免消息过长，钉钉截掉
		if count >= lineInOneMes {
			command.SendResult(temp, "")
			temp = ""
			count = 0
		}
	}
	models.ClientHotFixedDataLock.Unlock()
	result += temp
	result += fmt.Sprintf("\ntotalsize:%dMB\nfiles.txtmd5：%s", totalSize/1048576, tool.CalcMd5(path.Join(projectPath, models.CLIENTLOCALRESPATH, models.CLIENTHOTFIXEDFILENAME)))
	return result, nil
}

//上传测试热更资源
func uploadHotfixRes2Test(command *models.AutoBuildCommand) (string, error) {
	//获取项目信息
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return "", err
	}
	svnProjectName := params[0]
	err, projectPath, _, _ := models.GetSvnProjectInfo(command.ProjectName, svnProjectName)
	if err != nil {
		return "", err
	}

	//从缓存中取，如果没有则提示重新获取更新列表
	models.ClientHotFixedDataLock.RLock()
	defer models.ClientHotFixedDataLock.RUnlock()
	var needUpdateHotfiexedDataList []*models.ClientHotFixedFileData
	ok := false
	if needUpdateHotfiexedDataList, ok = models.ClientHotFixedFileDataTempMap[svnProjectName]; !ok {
		return "", errors.New(fmt.Sprintf("获取热更缓存数据失败，请重新执行【%s】指令！", models.GetCommandNameByType(models.CommandType_PrintHotfixResList)))
	}

	if nil == needUpdateHotfiexedDataList || len(needUpdateHotfiexedDataList) <= 0 {
		return "", errors.New(fmt.Sprintf("缓存热更数据为空，请重新执行【%s】指令试试！", models.GetCommandNameByType(models.CommandType_PrintHotfixResList)))
	}

	//获取cdn配置
	err, cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret, _, resPaths := models.GetCdnData(command.ProjectName, svnProjectName)
	if err != nil {
		return "", err
	}

	//获取cdn对象
	cdnErr, cdnClient := GetCdnClient(cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret)
	if nil != cdnErr {
		return "", cdnErr
	}

	//上传一个文件
	testPath := resPaths[0]
	count := 0
	uploadSuccessMsg := "开始上传测试热更资源：\n"
	uploadFile := func(uploadFileName string) error {
		localFilesPath := path.Join(projectPath, models.CLIENTLOCALRESPATH, uploadFileName)
		remoteFilePath := path.Join(testPath, uploadFileName)
		err := cdnClient.UploadFile(localFilesPath, remoteFilePath)
		if nil == err {
			count++
			uploadSuccessMsg += fmt.Sprintf("上传%s成功\n", uploadFileName)
		} else {
			return errors.New(fmt.Sprintf("上传%s失败，原因%s\n", uploadFileName, err.Error()))
		}

		//避免消息过长，钉钉截掉
		if count >= lineInOneMes {
			command.SendResult(uploadSuccessMsg, "")
			uploadSuccessMsg = ""
			count = 0
		}
		return nil
	}

	//先上传所有热更文件
	for _, hotfiexedData := range needUpdateHotfiexedDataList {
		err := uploadFile(hotfiexedData.Name)
		if err != nil {
			return "", err
		}
	}
	delete(models.ClientHotFixedFileDataTempMap, svnProjectName)

	//再上传files.txt并返回md5值
	err = uploadFile(models.CLIENTHOTFIXEDFILENAME)
	if err != nil {
		return "", err
	}

	//输出结果
	result := uploadSuccessMsg
	result += "\nfiles.txtmd5：" + tool.CalcMd5(path.Join(projectPath, models.CLIENTLOCALRESPATH, models.CLIENTHOTFIXEDFILENAME))
	return result, nil
}

//上传正式热更资源
func uploadHotfixRes2Release(command *models.AutoBuildCommand) (string, error) {
	//获取分支名称
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return "", err
	}
	svnProjectName := params[0]

	//获取cdn配置
	err, cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret, _, resPaths := models.GetCdnData(command.ProjectName, svnProjectName)
	if nil != err {
		return "", err
	}

	//获取cdn对象
	cdnErr, cdnClient := GetCdnClient(cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret)
	if nil != cdnErr {
		return "", cdnErr
	}

	//获取服务器资源地址files.txt列表
	resFileList := make(map[string]string)
	for _, _path := range resPaths {
		if _path == "" {
			continue
		}
		remoteHotfixedFilesPath := path.Join(_path, models.CLIENTHOTFIXEDFILENAME)
		fileErr, files := cdnClient.DownFile(remoteHotfixedFilesPath)
		if nil != fileErr {
			return "", errors.New(fmt.Sprintf("获取files.txt文件失败，资源路径：%s,错误原因：%s", remoteHotfixedFilesPath, fileErr.Error()))
		}
		resFileList[_path] = string(files)
	}

	//从测试地址拷贝资源到正式地址
	testHotfixedFilesPath := resPaths[0]
	testFiles := resFileList[testHotfixedFilesPath]
	result := ""
	for resPath, files := range resFileList {
		if resPath == testHotfixedFilesPath {
			continue
		}

		//拷贝文件
		count := 0
		_SuccessResult := fmt.Sprintf("开始从%s拷贝资源到%s\n", testHotfixedFilesPath, resPath)
		copyFunc := func(fileName string) error {
			testFilesPath := path.Join(testHotfixedFilesPath, fileName)
			targetFilePath := path.Join(resPath, fileName)
			err := cdnClient.CopyFile(testFilesPath, targetFilePath)
			if nil == err {
				count++
				_SuccessResult += fmt.Sprintf("拷贝%s成功\n", fileName)
			} else {
				return errors.New(fmt.Sprintf("拷贝%s失败，原因%s\n", fileName, err.Error()))
			}

			//避免消息过长，钉钉截掉
			if count >= lineInOneMes {
				command.SendResult(_SuccessResult, "")
				_SuccessResult = ""
				count = 0
			}
			return nil
		}

		//比对出不一样的资源并拷贝
		needUpdateFiles := models.GetNeedUpdateDatas(testFiles, files)
		if len(needUpdateFiles) <= 0 {
			result += resPath + "没有需要更新的资源\n"
			continue
		}

		//拷贝资源
		for fileName, _ := range needUpdateFiles {
			err := copyFunc(fileName)
			if err != nil {
				return "", err
			}
		}

		//拷贝files.txt
		err := copyFunc(models.CLIENTHOTFIXEDFILENAME)
		if err != nil {
			return "", err
		}

		_SuccessResult += fmt.Sprintf("从%s拷贝资源到%s结束\n", testHotfixedFilesPath, resPath)
		_SuccessResult += "***************************************************************************************\n"
		command.SendResult(_SuccessResult, "")
	}
	result += "上传正式热更资源结束。"
	return result, nil
}

//备份热更资源
func backupHotfixRes(command *models.AutoBuildCommand) (string, error) {
	//获取svn工程名称
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return "", err
	}
	svnProjectName := params[0]

	//判断本地files.txt是否存在
	err, projectPath, _, _ := models.GetSvnProjectInfo(command.ProjectName, svnProjectName)
	if nil != err {
		return "", err
	}
	localFilesPath := path.Join(projectPath, models.CLIENTLOCALRESPATH, models.CLIENTHOTFIXEDFILENAME)
	if !tool.CheckFileIsExist(localFilesPath) {
		return "", errors.New("项目不存在files.txt文件")
	}

	//获取cdn配置
	err, cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret, backupPath, resPaths := models.GetCdnData(command.ProjectName, svnProjectName)
	if nil != err {
		return "", err
	}

	//判断备份目录是否有files.txt，没有则拷贝
	backupFilesPath := path.Join(backupPath, models.CLIENTHOTFIXEDFILENAME)
	if !tool.CheckFileIsExist(backupFilesPath) {
		//不存在则拷贝最新文件，也不用备份了
		_, errCopy := tool.CopyFile(backupFilesPath, localFilesPath)
		if nil == errCopy {
			return fmt.Sprintf("不存在%s文件,已拷贝本地最新files.txt\n", backupFilesPath), nil
		} else {
			return "", errors.New(fmt.Sprintf("不存在%s文件且拷贝本地文件失败，错误原因：%s\n", backupFilesPath, errCopy.Error()))
		}
	}

	//获取本地files.txt文件
	result := ""
	backupFilesFile, err := os.Open(backupFilesPath)
	if err != nil {
		fmt.Println("读取备份file.txt失败 os.Open:", err)
		return "", errors.New("读取备份file.txt失败 os.Open:" + err.Error())
	}
	backupFilesFileByts, err := ioutil.ReadAll(backupFilesFile)
	backupFilesFile.Close()
	if err != nil {
		return "", errors.New("读取备份file.txt失败 ioutil.ReadAll:" + err.Error())
	}

	//获取cdn对象
	cdnErr, cdnClient := GetCdnClient(cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret)
	if nil != cdnErr {
		return "", cdnErr
	}

	//获取服务器正式热更files.txt
	remoteFormalHotfixedFilesPath := path.Join(resPaths[1], models.CLIENTHOTFIXEDFILENAME)
	formalFileErr, formalFiles := cdnClient.DownFile(remoteFormalHotfixedFilesPath)
	if nil != formalFileErr {
		return "", errors.New(fmt.Sprintf("获取正式files.txt文件失败，资源路径：%s,错误原因：%s", remoteFormalHotfixedFilesPath, formalFileErr.Error()))
	}

	//更新本地备份文件
	commandName := "sh"
	if runtime.GOOS == "windows" {
		commandName = winGitPath
	}
	svnUpdateCommand := fmt.Sprintf("cd %s;svn up .;", backupPath)
	svnUpdateResult, _ := tool.Exec_shell(commandName, svnUpdateCommand)
	result += svnUpdateResult

	//根据本地和服务器files。txt文件比对获取需要下载的数据
	needDownloadHotfixedFilesDataMap := models.GetNeedUpdateDatas(string(backupFilesFileByts), string(formalFiles))

	//下载跟本地不一样的文件
	for k, _ := range needDownloadHotfixedFilesDataMap {
		remoteFilePath := path.Join(resPaths[1], k)
		downloadedFileName := path.Join(backupPath, k)
		cdnClient.DownFile2Local(remoteFilePath, downloadedFileName)
	}

	//再下载files.txt
	cdnClient.DownFile2Local(remoteFormalHotfixedFilesPath, backupFilesPath)

	//上传svn
	svnLog := fmt.Sprintf("md5:%s \n", tool.CalcMd5(backupFilesPath))
	requestParams := strings.Split(command.CommandParams, ",")
	if len(requestParams) > 2 {
		svnLog += requestParams[1]
	}
	commitCommand := fmt.Sprintf("cd %s;chmod +x svnCommit.sh;./svnCommit.sh %s %s", shellPath, backupPath, svnLog)
	commitResult, _ := tool.Exec_shell(commandName, commitCommand)
	result += commitResult
	result += "备份完毕！"
	return result, nil
}

//更新用户组
func updateUserGroupCommand(command *models.AutoBuildCommand) (result string, err error) {
	userGroupInfo := command.CommandParams
	if strings.Contains(userGroupInfo, "{") {
		//更新用户数据
		return models.UpdateUserGroup(userGroupInfo)
	} else {
		//查询用户组数据
		return models.QueryUserGroupDatas(userGroupInfo), nil
	}
}

//更新用户
func updateUserCommand(command *models.AutoBuildCommand) (result string, err error) {
	userInfo := command.CommandParams
	if strings.Contains(userInfo, "{") {
		//更新用户数据
		return models.UpdateUserInfo(command.ProjectName, userInfo), nil
	} else {
		//查询用户数据
		return models.QueryUsersDatas(command.ProjectName, userInfo), nil
	}
}

//列出所有日志
func listAllSvnLog(command *models.AutoBuildCommand) (string, error) {
	//获取项目配置
	err, params := models.AnalysisParam(command.CommandParams, command.CommandType)
	if nil != err {
		return "", err
	}
	svnProjectName := params[0]
	err, _, svnPath, _ := models.GetSvnProjectInfo(command.ProjectName, svnProjectName)
	if nil != err {
		return "", err
	}

	//判断是否有时间参数
	startTimeStamp := models.GetSvnLogTime(command.ProjectName, svnProjectName)
	endTimeStamp := int64(0)
	if len(params) > 1 {
		timeStr := params[1]
		timeStrs := strings.Split(timeStr, "-")
		if len(timeStrs) >= 2 {
			startTimeStamp, _ = strconv.ParseInt(timeStrs[0], 10, 64)
			endTimeStamp, _ = strconv.ParseInt(timeStrs[1], 10, 64)
		} else {
			startTimeStamp, _ = strconv.ParseInt(timeStr, 10, 64)
		}
	}

	//获取svn日志的开始和截止日期参数
	var _startTime time.Time
	if startTimeStamp <= 0 {
		//默认10天前
		currentTime := time.Now()
		_startTime = currentTime.AddDate(0, 0, -10)
	} else {
		_startTime = time.Unix(int64(startTimeStamp), 0)
	}
	startDateStr := _startTime.Format("2006-01-02 15:04:05")
	endDateStr := "HEAD"
	if endTimeStamp > 0 {
		endDateStr = fmt.Sprintf("{'%s'}", time.Unix(int64(endTimeStamp), 0).Format("2006-01-02 15:04:05"))
	}

	commandName := "sh"
	if runtime.GOOS == "windows" {
		commandName = winGitPath
	}

	//获取svn路径日志
	commandTxt := fmt.Sprintf("svn log -r {'%s'}:%s %s", startDateStr, endDateStr, svnPath)
	log.Info(commandTxt)
	commandResult, _ := tool.Exec_shell(commandName, commandTxt)

	//解析该svn路径所有日志
	defaultLogType := "优化"
	defaultSysType := "unknowSystem"
	logFilters := strings.Split(logFilter, "|")
	ratLog := make(map[string]*models.SvnLog)
	svnLogs := strings.Split(commandResult, "------------------------------------------------------------------------")
	for _, v := range svnLogs {
		if v == "" {
			continue
		}

		//原始格式：版本、提交者、提交日期、内容以|分割
		logItem := strings.Split(v, "|")
		if len(logItem) < 4 {
			continue
		}
		//version := logItem[0]
		author := logItem[1]
		//date := logItem[2]

		//解析日志内容，原始格式：行数\r\n日志内容
		logItem[3] = strings.ReplaceAll(logItem[3], "\r", "")
		content := strings.Split(logItem[3], "\n")
		if len(content) < 4 {
			continue
		}

		//处理日志
		for i := 2; i < len(content); i++ {
			_logContent := content[i]
			if _logContent == "" {
				continue
			}

			//过滤掉不需要的日志
			isFilter := false
			for _, logFil := range logFilters {
				if strings.Contains(_logContent, logFil) {
					isFilter = true
					break
				}
			}
			if isFilter {
				continue
			}

			//获取日志的类型和系统
			temps := strings.Split(_logContent, "#")
			logType := defaultLogType
			logSysType := defaultSysType
			oneLineLog := _logContent
			if len(temps) > 2 {
				logType = temps[0]
				logSysType = temps[1]
				oneLineLog = ""
				for i := 2; i < len(temps); i++ {
					//#前添加转义字符，分割后这一行就是转移字符，所以替换回#号
					if temps[i] == "\\" {
						oneLineLog += "#"
					} else {
						oneLineLog += temps[i]
					}
				}
			}

			//初始化svn日志数据结构
			logType = strings.ToLower(logType)
			if _, ok := ratLog[logType]; !ok {
				//不存在该提交类型
				ratLog[logType] = new(models.SvnLog)
				ratLog[logType].LogType = logType
				ratLog[logType].Logs = make(map[string]map[string][]string)
			}
			if _, ok := ratLog[logType].Logs[logSysType]; !ok {
				//不存在该系统类型
				ratLog[logType].Logs[logSysType] = make(map[string][]string)
			}
			if _, ok := ratLog[logType].Logs[logSysType][author]; !ok {
				//不存在该提交作者
				ratLog[logType].Logs[logSysType][author] = make([]string, 0)
			}

			//添加日志内容
			ratLog[logType].Logs[logSysType][author] = append(ratLog[logType].Logs[logSysType][author], oneLineLog)
		}
	}

	//将该svn路径日志以一定的格式构建群
	svnLog := ""
	for _, logOfOneType := range ratLog {
		//按日志类型整理日志
		index := 0
		svnLog += fmt.Sprintf("%s:\n", logOfOneType.LogType)
		for logSysType, LogMapOfSysType := range logOfOneType.Logs {
			//按系统模块整理日志
			for author, logMapOfAuthor := range LogMapOfSysType {
				//按每个提交者整理日志
				for _, log := range logMapOfAuthor {
					index++
					svnLog += fmt.Sprintf("  %d、%s(%s by %s)\n", index, log, logSysType, author)
				}
			}
		}
	}
	svnLog += "\n"
	result := fmt.Sprintf("从%s到%s的svn日志:\n%s", startDateStr, endDateStr, svnLog)

	//保存svn截止日期（后面命令没有参数就是默认从这个时间到最新日志）
	if endTimeStamp <= 0 {
		endTimeStamp = time.Now().Unix()
	}
	models.SaveSvnLogTime(command.ProjectName, svnProjectName, endTimeStamp)
	return result, nil
}

//更新构建版本号
func UpdateBuildVerson(command *models.AutoBuildCommand) (result string) {
	buildVersionInfo := command.CommandParams
	if buildVersionInfo == "" {
		//如果为空则返回构建版本号
		result += "更新版本号以打包命令枚举和版本号以逗号分割，多个则以分号分割，如设置打安卓QC包版本号为1则：【更新构建版本号：8,1】\n"
		result += models.GetAllBuildVersionInfo()
	} else {
		//更新构建版本号
		models.SaveBuildVersion(buildVersionInfo)
		result = "更新构建版本号成功"
	}
	return
}

//更新服务进程配置
func updateCmdTempConfigCommand(command *models.AutoBuildCommand) (string, error) {
	svrConfig := command.CommandParams
	if strings.Contains(svrConfig, "{") {
		//更新svrProgress配置
		return models.UpdateCmdTemp(command.ProjectName, svrConfig), nil
	} else {
		//搜索svrProgress配置
		return models.QueryCmdTempsDataByProject(command.ProjectName, svrConfig), nil
	}
}

//备份svn
func backupSvn(command *models.AutoBuildCommand) (string, error) {
	count := 0
	temp := ""
	lastTime := time.Now().Unix()
	svnRepositories := ""
	startFlag := "backup "
	err := tool.RemoteShell("python backupSvn.py", svnMachineAccount, svnMachinePwd, svnMachineAddr, func(resultLine string) {
		//每隔80行发送一条构建消息
		if strings.Contains(resultLine, startFlag) {
			svnRepositories = strings.ReplaceAll(resultLine, "\r\n", "")
			resultLine = "start " + resultLine
		} else {
			resultLine = svnRepositories + resultLine
		}

		count++
		temp += resultLine
		timeNow := time.Now().Unix()
		if count >= lineInOneMes * 10 || (timeNow-lastTime) > maxIntervalTimeBetween2Msg {
			command.SendResult(temp, "")
			temp = ""
			count = 0
			lastTime = timeNow
		}
	})
	if nil == err {
		return temp, nil
	}
	return "", err
}

//空方法
func nullCommandFunc(command *models.AutoBuildCommand) (string, error) {
	return "", errors.New("没有实现的方法，不应该走到这里：" + command.Name)
}
