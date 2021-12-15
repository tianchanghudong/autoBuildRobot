#!/bin/bash

cd $1
for file in `svn status|grep "^ *?"|sed -e 's/^ *? *//'`; do rm -rf $file ; done
svn revert -R .
svn up .
svn merge $2 . --accept=$3
#外链不能被修改
svn revert ./Assets/LuaFramework/Lua
svn ci ./ -m "$4"
