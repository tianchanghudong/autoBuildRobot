#!/bin/bash

dirname=$1
targetdir="/data/"${dirname}
backfile=${dirname}".tgz"
excludedir=${dirname}"/logdata"
excuteFile="startAll.sh"

cd /data
tar -czvf ${backfile} --exclude=${excludedir} ${dirname}
cd /data/dus/server_file
unzip -o -d ${targetdir} ${dirname}".zip"

cd ${targetdir}
if [ ! -x "$excuteFile" ]; then
./stop.sh;sleep 1;./start.sh
else
./stopAll.sh;sleep 1;./startAll.sh
fi


