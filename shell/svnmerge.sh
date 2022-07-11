#!/bin/bash
cd $1

#还原本地、更新、合并
for file in `svn st|grep "^ *?"|sed -e 's/^ *? *//'`; do rm -rf $file ; done
svn revert -R .
svn up .
svn merge $2 . --accept=$3

#外链不能修改
isExistExternal=$(svn st | awk '{if ($1 == "X") {print $2} }' | sed 's/\\/\//g' | awk -F / '{for(i=1;i<NF;i++){if(i==NF-1){print $i "/"}else{printf $i "/"}}}')
if [ "$isExistExternal" != "" ];then 
svn st | awk '{if ($1 == "X") {print $2} }' | sed 's/\\/\//g' | awk -F / '{for(i=1;i<NF;i++){if(i==NF-1){print $i "/"}else{printf $i "/"}}}'|xargs svn revert
fi

#提交
svn ci ./ -m "$4"
