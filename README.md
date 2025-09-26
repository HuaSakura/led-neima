# 项目结构说明 (led-neima)

本项目是一个 **Go 语言项目**，主要功能可能与 LED 控制/播放相关。
此分支根据官方文档，针对与windows系统开发，运行不同的环境需要自行切换DLL文件夹下的库文件

项目目录结构如下：

```
led-neima/
├── bin/                 # 编译后的可执行文件目录
│   └── led_neima        # 主程序编译输出的二进制文件
│
├── DLL/                 # Windows 动态链接库目录
│   ├── 32/              # 32 位 DLL 库
│   │   ├── lv_led_32.dll         # 32 位，Unicode 字符集
│   │   └── lv_led_MBCS_32.dll    # 32 位，多字节字符集 (MBCS)
│   │
│   └── 64/              # 64 位 DLL 库
│       ├── lv_led_64.dll         # 64 位，Unicode 字符集
│       └── lv_led_MBCS_64.dll    # 64 位，多字节字符集 (MBCS)
│
├── firmware/            # 固件文件目录
│   └── T4-7.8G9.cpu     # T4固件文件
│
├── tools/               # 辅助工具目录
│   └── 内码协议.exe      # 协议相关工具 (Windows 可执行文件)
│
├── .gitignore           # Git 忽略规则文件
├── build.cmd            # Windows 下的构建脚本
├── go.mod               # Go Modules 依赖配置文件
├── main.go              # 项目入口文件 (Go 语言主程序)
└── README.md            # 项目说明文档
```
