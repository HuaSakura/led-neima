# 项目结构说明 (led-neima)

本项目是一个 **Go 语言项目**，主要功能可能与 LED 控制/播放相关。
此分支根据官方文档，针对与Linux系统开发，运行不同的环境需要自行切换SO文件夹下的库文件

项目目录结构如下：

```
led-neima/
├── bin/                 # 编译后的可执行文件目录
│   └── led_neima        # 主程序编译输出的二进制文件
│
├── firmware/            # 固件文件目录
│   └── T4-7.8G9.cpu     # T4固件文件
│
├── SO/                  # 动态链接库 (Shared Object) 文件目录
│   ├── arm/             # ARM 架构的库文件
│   │   ├── 32/          # ARM 32 位
│   │   │   └── libledplayer7.so
│   │   └── 64/          # ARM 64 位
│   │       └── libledplayer7.so
│   │
│   └── x86_64/          # x86_64 架构的库文件
│       └── libledplayer7.so
│
├── build.sh             # 构建脚本 (自动化编译、安装、部署用)
├── go.mod               # Go Modules 依赖配置文件
├── main.go              # 项目入口文件 (Go 语言主程序)
└── README.md            # 项目说明文档
```

