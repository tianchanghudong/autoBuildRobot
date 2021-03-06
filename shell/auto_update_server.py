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
import subprocess

def exe_command(command):
    try:
        sys.stdout.flush()
        p=subprocess.Popen(command, shell=True, stdout=sys.stdout,stderr=sys.stdout)
        result = p.wait()
        if result == 0:
            return True
        else:
            return False
    except Exception as e:
        print(e)
        return False

def svn_update(base_path):
    if exe_command('cd {0};svn update .'.format(base_path)):        
        print "svn update ok"
    else:
        print "svn update Exception"
        exit(2)

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
    print("gopath:"+os.environ.get("GOPATH"))
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
            #后面条件分支貌似要合并，但是貌似现在这样更可读
            elif file.find(svrProgressProjDirName) >= 0: 
                #根据平台不同，编译的可执行文件后缀也不一样
                if platform == "windows" and file.find("exe") >= 0:
                    print("windows，不存在文件：{0}，请检查".format(file))
                    return False 
                elif platform != "windows" and file.find("exe") < 0:
                    print("other platform,不存在文件：{0}，请检查".format(file))
                    return False 
            elif file != "":
                print("不存在文件：{0}，请检查".format(file))
                return False            
    f.close()
    print("zip OK")
    return True

def compile(platform):
    if exe_command("CGO_ENABLED=0 GOOS={0} GOARCH=amd64 go build".format(platform)):
        print("compile OK")
        return True
    else:
        print("compile failed")
        return False

def scp_upload(upload_ip,port,account,psd,file,uploadPath):
	ssh = paramiko.SSHClient()
	ssh.load_system_host_keys()
	ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
	ssh.connect(upload_ip,port,account,psd,timeout=10)
	
	with contextlib.closing(scpclient.Write(ssh.get_transport(), uploadPath)) as scp:
		scp.send_file(file, True, file)
	print("scp upload OK")

def update_svr(upload_ip,port,account,psd,platform,svrRootPath, zipFileName):
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(upload_ip,port,account,psd,timeout=10)
    sshCommand = 'cd {0};chmod +x mvandrestart_.sh;./mvandrestart_.sh {0} {1}'.format(svrRootPath,zipFileName)
    if platform == "windows":
        #windows烦死了，，macssh到window切换盘符以及执行多条命令烦死了，干脆直接把脚本放到windows的用户下面，直接执行脚本
        sshCommand = 'mvandrestart_.sh {0} {1}'.format(svrRootPath,zipFileName)
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
    scp_upload(upload_ip,port,account,psd,zipFileNameWithoutExt + ".zip",svrRootPath)
    
    #更新    
    update_svr(upload_ip,port,account,psd,platform,svrRootPath, zipFileNameWithoutExt)
    exit(0)