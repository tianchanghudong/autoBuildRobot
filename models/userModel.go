package models

import (
	"autobuildrobot/tool"
	"fmt"
	"github.com/astaxie/beego"
	"math"
	"strconv"
	"strings"
	"sync"
)

var usersMap map[string]*UserModel    //用户数据，key：用户昵称 value:用户数据
var lastProjectUserFileName = ""      //上一次的用户数据文件名（基本一个项目一个文件）
var superUser string;
var userDataLock sync.Mutex

//用户数据
type UserModel struct {
	Permission         int
	PhoneNum           string
	ProjectPermissions map[string]bool   //项目分支权限
}

func init(){
	temp, _ := beego.GetConfig("String", "superUser", "")
	superUser = temp.(string)
}

//获取项目管理员电话
func GetProjectManagerPhone(projectName string) string {
	userDataLock.Lock()
	defer userDataLock.Unlock()

	manager := GetProjectManager(projectName)
	if manager == "" {
		return ""
	}
	phone := ""
	managers := strings.Split(manager,"|")
	for _,_manager := range managers{
		_,users := getProjectUsersData(projectName)
		if user, ok := users[_manager]; ok {
			phone += user.PhoneNum + ","
		}
	}

	return phone
}

//判断是否有权限
func JudgeIsHadPermission(commandType int, projectName,useName,commandParams string) (bool,string) {
	//管理员有所有权限
	if strings.Contains(superUser, useName) || JudgeIsManager(projectName,useName){
		return true,""
	}

	//判断指令是否被禁止了
	if JudgeCommandIsBan(projectName,useName,GetCommandNameByType(commandType)){
		return false,"指令已被管理员禁止，联系管理员！"
	}

	//获取权限
	userDataLock.Lock()
	defer userDataLock.Unlock()
	permission := 0
	projectPermissions := make(map[string]bool)
	_,users := getProjectUsersData(projectName)
	if user, ok := users[useName]; ok {
		permission = user.Permission
		projectPermissions = user.ProjectPermissions
	}


	//判断是否有操作权限
	if !tool.Tool_BitTest(permission, uint(commandType+1)) {
		return false,"没有指令权限，请联系管理员！"
	}

	if JudgeIsHelpParam(commandParams){
		//如果是帮助，则只要判断指令权限就够了
		return true,""
	}

	//获取分支名称
	svnProject1, svnProject2,err := GetSvnProjectName(commandParams,commandType)
	if nil != err{
		return false,err.Error()
	}

	//不用判断是否有项目权限
	if svnProject1 == "" {
		return true,""
	}

	if commandType == CommandType_SvnMerge {
		//合并主要看有没有目标分支权限
		svnProject1 = svnProject2
	}

	//判断是否有项目权限
	if nil == projectPermissions {
		return false,"没有项目权限，请联系管理员！"
	}
	if _,ok := projectPermissions[svnProject1];ok{
		return true,""
	}else{
		return false,"没有项目权限，请联系管理员！"
	}
}

//获取用户电话
func GetUserPhone(projectName,_senderNick string) (phoneNum string) {
	userDataLock.Lock()
	defer userDataLock.Unlock()
	_,users := getProjectUsersData(projectName)
	if user, ok := users[_senderNick]; ok {
		phoneNum = user.PhoneNum
	}
	return
}

//获取所有用户信息
func GetAllUserInfo(projectName string) string {
	userDataLock.Lock()
	defer userDataLock.Unlock()
	_,users := getProjectUsersData(projectName)
	if len(users) <= 0 {
		return "当前没有任何用户，请添加:" + GetUserConfigHelp()
	}
	result := "\n***********************以下是已有的用户配置***********************\n"
	for k, v := range users {
		permissions := ""
		for i := 0; i < int(CommandType_Max); i++ {
			if tool.Tool_BitTest(v.Permission, uint(i+1)) {
				permissions += GetCommandNameByType(i) + "|"
			}
		}
		projectPermission := ""
		for _projectPermission,_ :=range v.ProjectPermissions {
			projectPermission += _projectPermission + "|"
		}
		result += fmt.Sprintf("%s,拥有指令权限：%s\n拥有项目权限：%s\n", k, permissions,projectPermission)
	}
	return result
}

//获取更新用户帮助
func GetUserConfigHelp()string{
	return fmt.Sprintf("例：【%s：用户名,手机号,拥有权限的指令序号1|指令序号2,拥有权限的分支名1|分支名2】\n参数依次为用户名手机号指令权限分支权限，参数间用英文逗号分割，权限间用|分割，多个用户用英文分号分割",commandName[CommandType_UpdateUser])
}

//更新用户数据
func UpdateUserInfo(projectName,userInfo string) (result string) {
	userDataLock.Lock()
	defer userDataLock.Unlock()
	var fileName string
	fileName, usersMap = getProjectUsersData(projectName)
	userArr := strings.Split(userInfo, ";")
	for _, user := range userArr {
		if user == "" {
			continue
		}
		userInfos := strings.Split(user, ",")
		if len(userInfos) < 4 {
			result = "输入信息不合法，名字电话权限项目权限以英文逗号分割，如张三,158xxx,0,xx项目"
			return
		}
		name := userInfos[0]
		phone := userInfos[1]
		permission := userInfos[2]
		projectPermission := userInfos[3]
		if name == "" {
			result = "名字不能为空，名字电话权限项目权限以英文逗号分割，如张三,158xxx,0,xx项目"
			return
		}
		if phone == "" && permission == "" && projectPermission == ""{
			//只有名字则删除用户
			delete(usersMap, name)
			continue
		}
		user := new(UserModel)
		if _, ok := usersMap[name]; ok {
			user = usersMap[name]
		}else{
			user.ProjectPermissions = make(map[string]bool)
		}
		if phone != "" {
			user.PhoneNum = phone
		}
		if permission != "" {
			//处理权限,用|分割拥有权限的枚举
			permissions := strings.Split(permission, "|")
			for _, v := range permissions {
				nPermission, _ := strconv.Atoi(v)
				if nPermission >= 0 {
					user.Permission = tool.Tool_BitSet(user.Permission, uint(nPermission+1))
				} else {
					//如果是负数则删除对应权限
					user.Permission = tool.Tool_BitClear(user.Permission, uint(math.Abs(float64(nPermission))+1))
				}
			}
		}
		if projectPermission != "" {
			//处理项目权限,用|分割拥有权限的项目名称
			projectPermissions := strings.Split(projectPermission, "|")
			for _, v := range projectPermissions {
				if strings.Contains(v,"-"){
					delete(user.ProjectPermissions,strings.ReplaceAll(v,"-",""))
				}else{
					user.ProjectPermissions[v] = true
				}
			}
		}
		usersMap[name] = user
	}

	//编码并存储
	tool.SaveGobFile(fileName, usersMap)
	result = "更新用户数据成功"
	return
}

//根据项目名获取项目用户数据
func getProjectUsersData(projectName string)(string,map[string]*UserModel){
	userDataFileName := "user.gob"
	fileName := ProjectName2Md5(projectName) + userDataFileName
	if fileName == lastProjectUserFileName {
		return fileName, usersMap
	}
	usersMap = make(map[string]*UserModel)
	tool.ReadGobFile(fileName, &usersMap)
	lastProjectUserFileName = fileName
	return fileName, usersMap
}