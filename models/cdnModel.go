package models

import (
	"autobuildrobot/log"
	"autobuildrobot/tool"
	"fmt"
	"strings"
	"sync"
)

//CDN配置
type CdnModel struct {
	ProjectName      string   `json:"ProjectName"`      //工程名称（对应SvnProjectModel工程名）
	CdnType          string   `json:"CdnType"`          //cdn类型,对应cdn.CdnType类型
	EndpointOfBucket string   `json:"EndpointOfBucket"` //用户Bucket所在数据中心的访问域名
	BucketName       string   `json:"BucketName"`       //Bucket名称
	AccessKeyID      string   `json:"AccessKeyID"`      //访问id
	AccessKeySecret  string   `json:"AccessKeySecret"`  //访问密钥
	BackupPath       string   `json:"BackupPath"`       //备份地址
	ResPaths         []string `json:"ResPaths"`         //资源地址（相对bucket根目录），默认第一个为测试地址

}

var lastProjectCdnFileName string      //上一次的CDN数据文件名（基本一个项目一个文件）
var projectCdnMap map[string]*CdnModel //项目CDN配置字典，key CDN名 value:项目CDN配置
var cdnDataLock sync.Mutex

const secretFlag = "***"

//有就更新，没有则添加
func UpdateCdn(projectName, cdnConfig string) (result string) {
	cdnDataLock.Lock()
	defer cdnDataLock.Unlock()

	//先获取数据
	var fileName string
	fileName, projectCdnMap = getProjectCdnsData(projectName)

	//再更新数据
	cdnArr := strings.Split(cdnConfig, ";")
	for _, cdn := range cdnArr {
		if cdn == "" {
			continue
		}
		cdnModel := new(CdnModel)
		tool.UnmarshJson([]byte(cdn), &cdnModel)
		if cdnModel.ProjectName == "" {
			errMsg := "cdn配置工程名不能为空：" + cdn
			log.Error(errMsg)
			result += (errMsg + "\n")
			continue
		}

		//删除配置
		if strings.Contains(cdnModel.ProjectName, "-") {
			//负号作为删除标记吧
			delBranch := strings.ReplaceAll(cdnModel.ProjectName, "-", "")
			delete(projectCdnMap, delBranch)
			continue
		}

		//判断工程是否存在
		if !JudgeSvnProjectIsExist(projectName, cdnModel.ProjectName) {
			errMsg := fmt.Sprintf("不存在%s工程，请先【更新svn工程配置】指令添加！\n", cdnModel.ProjectName)
			log.Error(errMsg)
			result += errMsg
			continue
		}

		//增加或修改
		if _cdnModel, ok := projectCdnMap[cdnModel.ProjectName]; ok {
			//已存在，如果数据为空则用老数据
			if cdnModel.CdnType == "" {
				cdnModel.CdnType = _cdnModel.CdnType
			}
			if cdnModel.EndpointOfBucket == "" {
				cdnModel.EndpointOfBucket = _cdnModel.EndpointOfBucket
			}
			if cdnModel.BucketName == "" {
				cdnModel.BucketName = _cdnModel.BucketName
			}
			if cdnModel.BackupPath == "" {
				cdnModel.BackupPath = _cdnModel.BackupPath
			}
			if cdnModel.AccessKeyID == "" || cdnModel.AccessKeyID == secretFlag {
				cdnModel.AccessKeyID = _cdnModel.AccessKeyID
			}
			if cdnModel.AccessKeySecret == "" || cdnModel.AccessKeySecret == secretFlag {
				cdnModel.AccessKeySecret = _cdnModel.AccessKeySecret
			}

			//处理资源地址
			mapPaths := make(map[string]string)
			for _, _oldPath := range _cdnModel.ResPaths {
				//保留老地址
				mapPaths[_oldPath] = _oldPath
			}
			for _, _path := range cdnModel.ResPaths {
				if strings.Contains(_path, "-") {
					//删除地址
					delPath := strings.Replace(_path, "-", "", 1)
					delete(mapPaths, delPath)
					continue
				}
				mapPaths[_path] = _path
			}
			cdnModel.ResPaths = make([]string, 0)
			for _, path := range mapPaths {
				//重新赋值所有地址
				cdnModel.ResPaths = append(cdnModel.ResPaths, path)
			}
		}
		projectCdnMap[cdnModel.ProjectName] = cdnModel
	}

	//编码并存储
	tool.SaveGobFile(fileName, projectCdnMap)
	if result != "" {
		return
	}
	return "更新cdn配置成功"
}

//获取一个项目所有CDN配置信息
func GetAllCdnDataOfOneProject(projectName string) string {
	cdnDataLock.Lock()
	defer cdnDataLock.Unlock()
	_, projectCdnMap = getProjectCdnsData(projectName)
	if len(projectCdnMap) <= 0 {
		return "当前没有cdn配置信息，请配置：\n" + GetCdnConfigHelp()
	}

	result := "\n***********************以下是已有的cdn配置***********************\n"
	tpl := CdnModel{}
	for _, v := range projectCdnMap {
		tpl.ProjectName = v.ProjectName
		tpl.CdnType = v.CdnType
		tpl.EndpointOfBucket = v.EndpointOfBucket
		tpl.BucketName = v.BucketName
		tpl.BackupPath = v.BackupPath
		tpl.ResPaths = v.ResPaths
		tpl.AccessKeyID = secretFlag
		tpl.AccessKeySecret = secretFlag
		result += fmt.Sprintln(tool.MarshalJson(tpl) + "\n")
	}
	return result
}

//获取cdn配置帮助提示
func GetCdnConfigHelp() string {
	tpl := CdnModel{
		ProjectName: "对应svn工程配置中得工程名称",
		CdnType:    "0:阿里云，1：华为云",
		BackupPath: "热更备份地址",
		ResPaths:   []string{"第一个默认为测试地址，地址都为Bucket下得相对路径，且不能有反斜杠用/", "路径2开始为正式地址1，多个地址后面追加"},
	}
	return fmt.Sprintf("例：\n【%s：%s】 \n如多个配置用英文分号分割", commandName[CommandType_UpdateCdnConfig],tool.MarshalJson(tpl))
}

//获取CDN配置数据
func GetCdnData(projectName, branchName string) (
	result, cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret,backupPath string, resPaths []string) {
	cdnDataLock.Lock()
	defer cdnDataLock.Unlock()
	if branchName == "" {
		result = "获取cdn地址失败，分支名不能为空！"
		return
	}
	_, projectCdnMap = getProjectCdnsData(projectName)
	if _branchModel, ok := projectCdnMap[branchName]; ok {
		return "", _branchModel.CdnType, _branchModel.EndpointOfBucket, _branchModel.BucketName,
			_branchModel.AccessKeyID, _branchModel.AccessKeySecret,_branchModel.BackupPath, _branchModel.ResPaths
	} else {
		result = "cdn配置不存在，请添加！"
		log.Error(branchName, result)
		return
	}
	return
}

//根据项目名获取cdn文件名和数据
func getProjectCdnsData(projectName string) (string, map[string]*CdnModel) {
	cdnDataFileName := "cdn.gob"
	fileName := ProjectName2Md5(projectName) + cdnDataFileName
	if fileName == lastProjectCdnFileName {
		return fileName, projectCdnMap
	}
	projectCdnMap = make(map[string]*CdnModel)
	tool.ReadGobFile(fileName, &projectCdnMap)
	lastProjectCdnFileName = fileName
	return fileName, projectCdnMap
}
