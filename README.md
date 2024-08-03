# WinSetup 项目简介

此项目是一个用于Windows并发安装软件包的Go程序，读取配置文件（config.toml），根据配置进行软件包的安装、升级等操作。程序通过并发处理加快安装速度，并且具备处理安装过程中常见问题的功能。

一次配置，终身使用，从此再也无惧重装系统后安装软件和环境的痛苦。

# 功能特性

- 支持并发安装，提升效率
- 自动跳过已安装的软件
- 可选参数：忽略安全哈希检查、跳过升级、卸载先前版本
- 通过配置文件灵活管理安装任务

# 安装

### 安装依赖程序

- winget: 参考官方 https://learn.microsoft.com/en-us/windows/package-manager/winget/

### 安装本程序

```
go install github.com/atopx/winsetup@latest
```

# 使用

### 编写`toml`配置文件

> 我的真实配置: [config.example.toml](./config.toml)

```toml
# 并发安装的协程数量, 建议1-8之间
concurrent = 4

# 路径别名
[location]
app = "C:\\apps"
lib = "C:\\libs"

# 这个例子实际安装位置: C:\apps\vscode
[[target]]
# 必选参数, 如果不知道软件ID, 可以使用 winget search, 比如: winget search vscode
id = "Microsoft.VisualStudioCode"
# link 基础安装路径, 程序会从`[location]`下找对应的路径链接，找不到会使用默认路径安装, 可选
link = "app"
# path 在基础路径上的拼接路径
# 可选、默认根据ID自动转换: "Microsoft.VisualStudioCode" => "microsoft\visualstudiocode"
path = "vscode"
# no-upgrade 如果已安装的版本已存在，则跳过升级, 可选、默认false
no-upgrade = true
# ignore-security-hash 忽略安装程序哈希检查失败, 可选、默认false
ignore-security-hash = false
# uninstall-previous 升级期间卸载以前版本的程序包, 可选、默认false
uninstall-previous = false

# 这个例子实际安装位置: C:\libs\golang\go
[[target]]
id = "Golang.Go"
link = "lib"
```

### 执行

```sh
winsetup.exe -c config.toml -n 2
```

##### 参数说明

```
Usage of winsetup.exe:
  -c config.toml
        配置路径, 默认当前路径的config.toml (default "config.toml")
  -n 4
        并发安装的协程数量, 默认4 (default 4)
```

### 错误处理
程序运行过程中，如果遇到错误，会记录到日志中并继续处理其他任务。确保正确配置 config.toml 以减少错误的发生。

# 贡献
欢迎提交PR和Issue，如果有任何问题或建议，请联系项目维护者。

# 许可证
此项目基于MIT许可证开源。详见LICENSE文件。
