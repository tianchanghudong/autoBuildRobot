package models

import (
	"strings"
	"sync"
)

const CLIENTHOTFIXEDFILENAME = "files.txt"          //热更资源列表文件名称
const CLIENTLOCALRESPATH = "Assets/StreamingAssets" //客户端本地资源地址

var ClientHotFixedFileDataTempMap map[string][]*ClientHotFixedFileData //客户端更新数据缓存(直接用，但是用时要加锁) key:分支名称  value:分支的热更数据缓存
var ClientHotFixedDataLock sync.RWMutex

//文件信息
type ClientHotFixedFileData struct {
	Name string //文件名
	MD5  string //文件MD5
	Size string //文件大小
}

func init(){
	ClientHotFixedFileDataTempMap = make(map[string][]*ClientHotFixedFileData)
}

//获取需要更新的文件数据
func GetNeedUpdateDatas(newFiles,oldFiles string)(needUpdateHotfixedFilesDataMap map[string]*ClientHotFixedFileData){
	newHotfixedFilesDataMap := AnslysisClientHotFixedFile(newFiles)
	oldHotfixedFilesDataMap := AnslysisClientHotFixedFile(oldFiles)

	//遍历file1文件列表，找到MD5不一致的，或者是file2不存在的文件
	needUpdateHotfixedFilesDataMap = make(map[string]*ClientHotFixedFileData)
	for fileName, newFileData := range newHotfixedFilesDataMap {
		if oldFileData, ok := oldHotfixedFilesDataMap[fileName]; ok {
			//存在的，但是MD5不一样的要上传
			if strings.Compare(oldFileData.MD5, newFileData.MD5) != 0 {
				needUpdateHotfixedFilesDataMap[fileName] = newFileData
			}
		} else {
			//不存在的要上传
			needUpdateHotfixedFilesDataMap[fileName] = newFileData
		}
	}
	return
}

//通过files.txt的[]byte转成改文件的文件数据
func AnslysisClientHotFixedFile(fileString string) map[string]*ClientHotFixedFileData {
	content := strings.Replace(fileString, "\r\n", "\n", -1)
	fileList := strings.Split(content, "\n")
	fileMap := make(map[string]*ClientHotFixedFileData)

	//解析files.txt
	for i := 0; i < len(fileList); i++ {
		oneFileDataStr := fileList[i]
		if strings.Compare(oneFileDataStr, "") == 0 {
			continue
		}
		arrTemp := strings.Split(oneFileDataStr, "|")
		fileMap[arrTemp[0]] = &ClientHotFixedFileData{Name: arrTemp[0], MD5: arrTemp[1], Size: arrTemp[2]}
	}

	return fileMap
}