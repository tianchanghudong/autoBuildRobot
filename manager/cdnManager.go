package manager

import (
	"autobuildrobot/manager/cdn"
	"errors"
	"strconv"
	"sync"
)

var cdnList map[string]cdn.IcdnClient
var cdnLock sync.RWMutex

func init() {
	cdnList = make(map[string]cdn.IcdnClient)
}

//获取cdn对象
func GetCdnClient(cdnType, urlOfBucket, bucketName, accessKeyID, accessKeySecret string) (error, cdn.IcdnClient) {
	cdnKey := cdnType + bucketName

	//从缓存中获取数据
	cdnLock.RLock()
	if icdnClient, ok := cdnList[cdnKey]; ok {
		cdnLock.RUnlock()
		return nil, icdnClient
	}
	cdnLock.RUnlock()

	//缓存中没有则新建实例
	switch cdnType {
	case strconv.Itoa(cdn.CdnType_ALiYun):
		{
			err, cdnClient := cdn.NewALiYun(urlOfBucket, bucketName, accessKeyID, accessKeySecret)
			if nil != err {
				return err, nil
			}
			cdnLock.Lock()
			defer cdnLock.Unlock()
			cdnList[cdnKey] = cdnClient
			return nil, cdnClient
		}
	case strconv.Itoa(cdn.CdnType_HuaWeiYun):
		{
			err, cdnClient := cdn.NewHuaWeiYun(urlOfBucket, bucketName, accessKeyID, accessKeySecret)
			if nil != err {
				return err, nil
			}
			cdnLock.Lock()
			defer cdnLock.Unlock()
			cdnList[cdnKey] = cdnClient
			return nil, cdnClient
		}
	default:
		return errors.New(cdnType + "cdn不存在"), nil
	}
}
