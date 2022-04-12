package models

import (
	"autobuildrobot/tool"
	"errors"
	"fmt"
	"strings"
	"sync"
)

//用户组
type UserGroupModel struct {
	GroupName          string   `json:"GroupName"`          //组名
	CommandPermissions []int    `json:"CommandPermissions"` //指令权限数组
	ProjectPermissions []string `json:"ProjectPermissions"` //项目权限数组
}

var userGroupFileName = "userGroupData.gob"
var userGroupMap map[string]*UserGroupModel
var userGroupDataLock sync.Mutex

func init() {
	userGroupMap = make(map[string]*UserGroupModel)
	tool.ReadGobFile(userGroupFileName, &userGroupMap)
}

//有就更新，没有则添加
func UpdateUserGroup(groupConfig string) (result string, err error) {
	userGroupDataLock.Lock()
	defer userGroupDataLock.Unlock()

	//解析数据
	groupModel := new(UserGroupModel)
	groupModel.CommandPermissions = make([]int, 0)
	groupModel.ProjectPermissions = make([]string, 0)
	err = tool.UnmarshJson([]byte(groupConfig), &groupModel)
	if nil != err {
		return
	}

	//编码并存储
	if strings.Contains(groupModel.GroupName, "-") {
		//表示删除
		groupName := strings.Replace(groupModel.GroupName, "-", "", 1)
		delete(userGroupMap, groupName)
	} else {
		//新增或更新
		userGroupMap[groupModel.GroupName] = groupModel
	}

	tool.SaveGobFile(userGroupFileName, userGroupMap)
	result = "更新用户组成功"
	return
}

//查找数据
func QueryUserGroupDatas(searchParams string) string {
	if JudgeIsSearchAllParam(searchParams) {
		return GetAllUserGroupInfo()
	} else {
		return GetUserGroupInfoByName(searchParams)
	}
}

//获取一个项目所有CDN配置信息
func GetAllUserGroupInfo() string {
	cdnDataLock.Lock()
	defer cdnDataLock.Unlock()

	if len(userGroupMap) <= 0 {
		return "当前没有任何用户组，请添加," + GetUserGroupConfigHelp()
	}
	result := "\n***********************以下是所有的用户组配置***********************\n"
	tpl := UserGroupModel{}
	for _, v := range userGroupMap {
		tpl.GroupName = v.GroupName
		tpl.CommandPermissions = v.CommandPermissions
		tpl.ProjectPermissions = v.ProjectPermissions
		result += fmt.Sprintln(tool.MarshalJson(tpl) + "\n")
	}
	return result
}

//获取项目配置数据
func GetUserGroupInfoByName(groupName string) (result string) {
	userGroupDataLock.Lock()
	defer userGroupDataLock.Unlock()
	if _group, ok := userGroupMap[groupName]; ok {
		return tool.MarshalJson(_group)
	}

	//如果不存在项目，则输出默认值
	return "用户组不存在，请添加：" + GetUserGroupConfigHelp()
}

//获取用户组帮助信息
func GetUserGroupConfigHelp() (result string) {
	group := new(UserGroupModel)
	group.GroupName = "用户组名"
	group.CommandPermissions = make([]int, 0)
	group.ProjectPermissions = make([]string, 0)
	group.ProjectPermissions = append(group.ProjectPermissions, "拥有权限的项目名1")
	group.ProjectPermissions = append(group.ProjectPermissions, "拥有权限的项目名2")
	return fmt.Sprintf("例：\n【%s：%s】\nCommandPermissions为拥有权限的指令索引数组\n如多个配置用分号分割",
		commandName[CommandType_UpdateUserGroup], tool.MarshalJson(group))
}

//获取指令权限
func GetUserGroupPermissions(groupName string) (error, []int, []string) {
	userGroupDataLock.Lock()
	defer userGroupDataLock.Unlock()
	if project, ok := userGroupMap[groupName]; ok {
		return nil, project.CommandPermissions, project.ProjectPermissions
	}
	return errors.New("用户组不存在,请配置！！！"), make([]int, 0), make([]string, 0)
}

//获取所有权限描述
func GetAllPermissionDesc(groupName string) (err error, commandPermission, projectPermission string) {
	userGroupDataLock.Lock()
	defer userGroupDataLock.Unlock()
	if project, ok := userGroupMap[groupName]; ok {
		for _, v := range project.CommandPermissions {
			commandPermission += GetCommandNameByType(v) + "|"
		}
		for _, v := range project.ProjectPermissions {
			projectPermission += v + "|"
		}
		return
	}
	return errors.New("用户组不存在,请配置！！！"), "", ""
}

//判断用户组是否存在
func JudgeUserGroupIsExist(groupName string) bool {
	userGroupDataLock.Lock()
	defer userGroupDataLock.Unlock()
	if _, ok := userGroupMap[groupName]; ok {
		return true
	}
	return false
}
