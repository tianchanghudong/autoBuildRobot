#!/bin/bash
svrRoot=$1
dirname=$2
zipfile=${dirname}".zip"
backfile=${dirname}".tgz"
excludedir=${dirname}"/logdata"
excuteFile="startAll.sh"

#如果文件夹不存在，则创建文件夹
cd ${svrRoot}
if [ ! -d "$dirname" ]; then
mkdir $dirname
fi

#备份
tar -czvf ${backfile} --exclude=${excludedir} --exclude=${zipfile} ${dirname}

#解压
unzip -o ${zipfile} -d ${dirname}

#后台是拷贝文件夹下压缩包，所以这里拷贝一下
sleep 2
mv -f ${zipfile} ${dirname}

#启动
cd ${dirname}
if [ ! -x "$excuteFile" ]; then
./stop.sh;sleep 5;./start.sh;exit
else
./stopAll.sh;sleep 10;./startAll.sh;exit
fi


