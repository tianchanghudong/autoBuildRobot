#!/bin/bash
svrRoot=$1
dirname=$2
zipfile=${dirname}".zip"
backfile=${dirname}".tgz"
excludedir=${dirname}"/logdata"
excuteFile="startAll.sh"

cd ${svrRoot}
tar -czvf ${backfile} --exclude=${excludedir} --exclude=${zipfile} ${dirname}
cd ${dirname}
unzip -o ${zipfile}

if [ ! -x "$excuteFile" ]; then
./stop.sh;sleep 5;start ./start.sh;exit
else
./stopAll.sh;sleep 10;./startAll.sh;exit
fi


