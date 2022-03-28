#!/usr/bin/python
# coding=utf-8

import os
import sys
import zipfile
import glob
import paramiko
import contextlib
import scpclient
import os.path

def svn_update(base_path):
    os.system('cd {0};svn update .'.format(base_path))
    print "svn update ok"

def zipdir(f, dirname):
    files = glob.glob('./%s/*'%dirname)
    for file in files:
        if os.path.isdir(file):
            zipdir(f, file)
        else:
            f.write(file, '%s/'%dirname + os.path.basename(file))

def package(projectPath,svrProgressProjDirName,platform,zipFileNameWithoutExt,zipDirList,zipFileList):
    GOPATH_ = projectPath + ":" + projectPath + "/package"
    os.environ["GOPATH"] = GOPATH_
    os.chdir(projectPath + "/src/" + svrProgressProjDirName)
    if compile(platform) == False:
        return False
    f = zipfile.ZipFile(zipFileNameWithoutExt + ".zip", 'w', zipfile.ZIP_DEFLATED) 
    
    #压缩文件夹
    if zipDirList != "":
        _zipDirList = zipDirList.split("|")
        for dir in _zipDirList:
            zipdir(f, dir)
    
    #压缩文件
    if zipFileList != "":
        _zipFileList = zipFileList.split("|")
        for file in _zipFileList:
            if os.path.exists(file):            
                f.write(file)
            #后面分支貌似要合并，但是貌似现在这样更可读
            elif file.find(svrProgressProjDirName) >= 0: 
                #根据平台不同，编译的可执行文件后缀也不一样
                if platform == "windows":
                    if file.find("exe"):
                        print("不存在文件：{0}，请检查".format(file))
                        return False 
                else:
                    print("不存在文件：{0}，请检查".format(file))
                    return False 
            else:
                print("不存在文件：{0}，请检查".format(file))
                return False            
    f.close()
    print("zip OK")
    return True

def compile(platform):
    compile_result = os.system("GOOS={0} GOARCH=amd64 go build".format(platform))
    if compile_result != 0:
        print("compile failed")
        return False
        
    print("compile OK")
    return True
    
def scp_upload(upload_ip,port,account,psd,file,uploadPath):
	ssh = paramiko.SSHClient()
	ssh.load_system_host_keys()
	ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
	ssh.connect(upload_ip,port,account,psd,timeout=10)
	
	with contextlib.closing(scpclient.Write(ssh.get_transport(), uploadPath)) as scp:
		scp.send_file(file, True, file)
	print("scp upload OK")

def update_svr(upload_ip,port,account,psd,platform,updateShPath, zipFileName):
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(upload_ip,port,account,psd,timeout=10)
    sshCommand = 'cd {0};./mvandrestart_.sh {1}'.format(updateShPath,zipFileName)
    if platform == "windows":
        #windows烦死了，，macssh到window切换盘符以及执行多条命令烦死了，干脆直接把脚本放到windows的用户下面，直接执行脚本
        sshCommand = 'mvandrestart_.sh {0}'.format(zipFileName)
    stdin, stdout ,stderr = ssh.exec_command(sshCommand)
    out = stdout.readlines()
    #for o in out:
	#	print o
    ssh.close()
    print("update OK")

if __name__ == '__main__':
    if len(sys.argv) < 12:
        print("not enough params:")
        exit(1)
    #所有参数
    projectPath = sys.argv[1]
    svrProgressProjDirName = sys.argv[2]
    platform = sys.argv[3]
    zipFileNameWithoutExt = sys.argv[4]
    zipDirList = sys.argv[5]
    zipFileList = sys.argv[6]
    upload_ip = sys.argv[7]
    port = sys.argv[8]
    account = sys.argv[9]
    psd = sys.argv[10]
    svrRootPath = sys.argv[11]
    
    #更新    
    svn_update(projectPath)
    
    #编译    
    if package(projectPath,svrProgressProjDirName,platform,zipFileNameWithoutExt,zipDirList,zipFileList) == False:
        exit(2)
    
    #上传
    uploadPath = os.path.join(svrRootPath,zipFileNameWithoutExt)
    scp_upload(upload_ip,port,account,psd,zipFileNameWithoutExt + ".zip",uploadPath)
    
    #更新
    updateShPath = os.path.join(svrRootPath, "dus")
    update_svr(upload_ip,port,account,psd,platform,updateShPath, zipFileNameWithoutExt)
    exit(0)