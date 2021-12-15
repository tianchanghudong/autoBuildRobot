# autobuildrobot
一、自动构建机器人，目前只接入了钉钉，支持拓展微信等
二、以钉钉为例，各个项目群可以分别加入钉钉机器人，每个群的标题是不一样的，通过标题区分各个项目进行分别管理
三、收到钉钉消息后解析处理再返回结果到钉钉群
四、实现的功能有：
1、项目配置：每个项目分前后端，如小精灵客户端、小精灵服务端（主要配置项目名称，管理员等）【projectModel.go】
2、分支配置：主要配置分支名称、svn地址、工程地址等【branchModel.go】
3、cdn配置：主要配置分支名称（对应分支配置数据）、cdn类型（阿里云、亚马逊等）、Bucket名等【cdnModel.go】
4、分支合并：合并流程看https://www.kdocs.cn/l/spWN1ZyWsEPr?f=131
5、更新和出包：更新客户端并出包
6、打热更资源：打客户端热更资源
7、输出热更资源列表：本地最新资源跟cdn服务器测试资源对比，列出本地修改的资源
8、上传测试热更资源：将要更新的资源上传到cdn测试地址
9、上传正式热更资源：测试地址资源验证没问题后上传到正式地址
10、更新并重启内网服务器
11、更新并重启外网测试服
12、打印svn日志：根据分支名称，输出该分支下的日志，格式 日志序列、日志内容（系统 by 修改人）
13、更新用户：管理自动构建用户，设置每个指令以及分支权限（各个项目独立设置）
14、代办事项提醒
15、更新表格：更新研发表格和正式表格，每个项目独立

