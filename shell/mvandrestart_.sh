#!/bin/bash
dirname=$1
zipfile=${dirname}".zip"
targetdir="/data/"${dirname}
backfile=${dirname}".tgz"
excludedir=${dirname}"/logdata"
excuteFile="startAll.sh"

cd /data
#tar -czvf ${backfile} --exclude=${excludedir} --exclude=${zipfile} ${dirname}
cd ${targetdir}
unzip -o ${zipfile}

if [ ! -x "$excuteFile" ]; then
./stop.sh;sleep 10;./start.sh
else
./stopAll.sh;sleep 10;./startAll.sh
fi


