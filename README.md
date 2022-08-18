# autobuildrobot
一、自动构建机器人，目前只接入了钉钉，支持拓展微信等
二、以钉钉为例，各个项目群可以分别加入钉钉机器人，每个群的标题是不一样的，通过标题区分各个项目进行分别管理
三、收到钉钉消息后解析处理再返回结果到钉钉群
四、实现的功能有：
1、项目配置：每个项目分前后端，如小精灵客户端、小精灵服务端（主要配置项目名称，管理员、客户端打包方法列表等）【projectModel.go】
2、svn工程配置：构建最核心配置，很多功能参数用到这里得配置，主要配置svn工程名称、svn地址、工程地址、外链关键字【svnProjectModel.go】
3、cdn配置：理解为热更需要的svn工程配置的补充配置，主要配置cdn名称（名称和svn工程配置相同，减少svn工程关联字段）、cdn类型（阿里云、亚马逊等）、Bucket名等【cdnModel.go】
4、检出svn工程：根据svn工程配置，存在则输出svn信息，不存在则检出svn工程到配置地址（因为操作耗时，所以就不再svn工程配置中判断操作，直接单独一个指令）
5、分支合并：目前前后端都定为5大分支，分别为临时开发分支（跨版本迭代），开发分支（常规迭代开发），策划分支（开发完成一个功能直接合并给策划验收），
测试分支（一个迭代所有功能策划验收完成后由策划分支合并到测试分支，测试分支会比开发晚一个迭代），发版分支（测试验收完毕合并到发版分支准备对外）
更详细流程看https://www.kdocs.cn/l/spWN1ZyWsEPr?f=131
6、更新表格：将表格分别输出客户端和服务器需要的lua和gob文件，前后端分别用svn外链引用，其中临时开发分支引用临时表格，开发和策划分支引用研发表格，
测试分支引用测试表格，发版分支引用正式表格
7、客户端自动构建：根据参数，执行打lua资源、打整个资源，出白包、以及各个渠道包
8、输出热更资源列表：根据参数，目标工程本地文件列表跟配置的cdn服务器比对，列出差异文件，用于看热更大小以及判断是否都是我们要热更的资源
9、上传测试热更资源：将要更新的资源上传到cdn测试地址，本地加白名单用正式包验证
10、上传正式热更资源：测试地址资源验证没问题后，定好维护时间，上传到正式地址
11、备份热更资源：本地维护完毕，备份下整个资源，用于下次热更如果出现意外需要回滚的资源备份
12、游戏服务进程配置：一套服务器有很多服务（如中心服、跨服、网关服、游戏服等），这个就是配置每个服务信息用于更新服务器【gameSvrProgressModel.go】
13：游戏服主机配置：理解为更新服务器需要的svn工程配置的补充配置，用于配置更新服务器的目标主机信息【svrMachineModel.go】
14：更新并重启服务器：如字面意思，流程是更新、编译、压缩、上传、备份、解压并重启服务器
15、打印svn日志：根据分支名称，输出该分支下的日志，格式 日志序列、日志内容（系统 by 修改人）
16：用户组配置：配置用户组权限等【userGroupModel.go】
17、用户配置：如题，配置用户信息（如名称 所属用户组等）【userModel.go】
五、机器人部署（钉钉）：
1、登录钉钉开放平台，选择应用开发-机器人-创建机器人
2、点击创建好的机器人，在凭证与基础信息中，将AppSecret复制到app.conf文件中的dingdingRobotAppSecret配置字段
2、在开发管理中，填上出口ip以及消息接收地址（一般机器人都是搭建在内网，所以要用内网穿透工具，填上外网能访问的地址）
3、在版本管理与发布中，选择发布上线，然后在公司群中找到智能群助手-添加上面创建的机器人即可
4、启动机器人程序，在上面的群中@机器人，看能否正常响应
六、机器人使用：
聊天框中@机器人，输入指令名称或者编号选择操作（如果输入得内容不匹配任何指令则会列出所有指令），指令后加冒号和参数如【更新表格：研发表格】
如果不清楚参数则输入帮助或者help会输出详细帮助提示如【更新表格：帮助】
指令如果不带动词，表示配置型指令，配置型指令参数为空则会输出所有已有数据或者输入查询条件筛选出对应数据
如果要执行多条指令，则指令间用->连接，如【更新表格：研发表格->更新表格：正式表格】
七、其他问题
1、如果依赖包下载超时，在终端执行go env -w GOPROXY=https://goproxy.cn


