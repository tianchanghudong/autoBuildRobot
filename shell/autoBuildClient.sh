#!/bin/sh
projectPath=$2
cd $projectPath
for file in `svn status|grep "^ *?"|sed -e 's/^ *? *//'`; do rm -rf $file ; done
svn revert -R .
svn up .
cd "$1"
./Unity.app/Contents/MacOS/Unity -quit -batchmode -projectPath $projectPath -executeMethod AutoBuild.StartBuild -buildMethod:$3 -webHook:$4
