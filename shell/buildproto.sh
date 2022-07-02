#!/bin/sh
#先更新
curDir=`cd $(dirname $0); pwd -P`
tarDir=$1
cd $tarDir
svn up .

#原始文件和生成文件路径
proto_pth=${tarDir}/proto
go_out_path=${tarDir}/go
lua_out_path=${tarDir}/lua

#生成pb文件
cd $curDir/proto
protoc -I=${proto_pth} --go_out=$go_out_path --plugin=protoc-gen-go=./protoc_gen_go_darwin ${proto_pth}/*.proto
protoc -I=${proto_pth} --lua_out=$lua_out_path --plugin=protoc-gen-lua=./protoc-gen-lua/plugin/protoc-gen-lua ${proto_pth}/*.proto


