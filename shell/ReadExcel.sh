#!/bin/sh
#更新策划表格
curDir=`cd $(dirname $0); pwd -P`
cd $1
svn up .

#用最近时间判断表格目录是否有更新
lastTime=`svn info|grep "最后修改的时间"|awk '{print $2,$3}'`
cd data
newLog=`svn log -r {"${lastTime}"}:HEAD|grep -v "-"`
if [ "${newLog}"="" ]
then
echo "表格已是最新，无需构建！"
exit
fi
cd ..

#windows平台下打表
_goos=`go env GOOS`
if [ $2 == "windows" ];then
${curDir}/ReadExcel.exe FanYi
./go1.12.1/bin/go.exe build GenerateGob.go
./GenerateGob.exe

#mac平台下打表
elif [ $2 == "darwin" ];then
chmod +x ${curDir}/ReadExcel
${curDir}/ReadExcel FanYi
go build GenerateGob.go
chmod +x GenerateGob
./GenerateGob

#其他平台
else
echo {$2}平台未实现导表功能，请添加
fi

#删除临时文件（mac还要第一行才能删除GenerateGob文件）
rm -rf GenerateGob
rm -rf GenerateGob.*
