package cdn

const (
	CdnType_ALiYun    = 0
	CdnType_HuaWeiYun = 1
	CdnType_Aws       = 2
	CdnType_Max
)

type IcdnClient interface {
	IsExistFile(remoteFilePath string) (bool,error)
	UploadFile(localFilePath, remoteFilePath string) error
	DownFile(remoteFilePath string) (err error, data []byte)
	DownFile2Local(remoteFilePath, downloadedFileName string) error
	DeleteFile(remoteFilePath string) error
	CopyFile(resPath, targetPath string) error
	Refresh(remoteFilePath string) error //预热
}
