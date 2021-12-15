#!/bin/sh
#更新策划表格
cd $1
svn up .

#windows平台下打表
_goos=`go env GOOS`
if [ ${_goos} == "windows" ];then
./ReadExcel.exe go/ data_manager lua_temp/ SSServer FanYi
./go1.12.1/bin/go.exe build GenerateGob.go
./GenerateGob.exe

#mac平台下打表
elif [ ${_goos} == "darwin" ];then
chmod +x ReadExcel
./ReadExcel go/ data_manager lua_temp/ SSServer FanYi
go build GenerateGob.go
chmod +x GenerateGob
./GenerateGob

#其他平台
else
echo {$_goos}平台未实现导表功能，请添加！
fi

#删除临时文件（mac还要第一行才能删除GenerateGob文件）
rm -rf GenerateGob
rm -rf GenerateGob.*

#提交修改
addFile=`svn st | awk '{if ($1 == "?") {print $2} }' | xargs`
if [ "$addFile" != "" ];then 
svn add $addFile
fi
delFile=`svn st | awk '{if ($1 == "!") {print $2} }' | xargs`
if [ "$delFile" != "" ];then 
svn del $delFile
fi
svn ci ./ -m "latest table!"
