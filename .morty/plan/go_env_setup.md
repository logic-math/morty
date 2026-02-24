# Plan: Go 开发环境搭建

## 模块概述

**模块职责**: 在开发机上搭建 Go 开发环境，包括 Go 安装、环境配置和必要工具安装

**对应 Research**:
- `morty-go-refactor-plan.md` 第 3 节技术栈: Go 1.21+

**依赖模块**: 无

**被依赖模块**: 所有 Go 模块 (CLI, Config, Logging, State, Git, Plan Parser, Executor)

---

## 目标环境

### 系统要求
- **操作系统**: Linux (CentOS/RHEL/Ubuntu) 或 macOS
- **架构**: x86_64 或 arm64
- **磁盘空间**: 至少 1GB 可用空间
- **网络**: 可访问互联网（下载 Go 和依赖）

### Go 版本
- **目标版本**: Go 1.21 或更高
- **安装路径**: `/usr/local/go` 或 `$HOME/go`
- **工作目录**: `$HOME/go`

---

## 接口定义

### 输入
- 系统环境（Linux/macOS）
- 用户权限（root 或普通用户）

### 输出
- 可用的 `go` 命令
- 配置好的 GOPATH 环境
- 必要的开发工具

---

## Jobs (Loop 块列表)

---

### Job 1: 系统依赖检查与安装

**目标**: 检查并安装系统级依赖

**前置条件**:
- 无

**Tasks (Todo 列表)**:
- [x] Task 1: 检查操作系统类型 (`uname -s`)
- [x] Task 2: 检查架构类型 (`uname -m`)
- [x] Task 3: 安装 wget 或 curl（用于下载）
- [x] Task 4: 安装 tar（用于解压）
- [x] Task 5: 检查 git 版本 >= 2.0
- [x] Task 6: 安装 make（用于构建）
- [x] Task 7: 验证所有依赖可用

**验证器**:
- [x] `uname` 命令返回正确结果 (Linux, x86_64)
- [x] `wget` 或 `curl` 可用 (/usr/bin/wget)
- [x] `tar` 命令可用 (/usr/bin/tar)
- [x] `git --version` >= 2.0 (git version 2.33.0)
- [x] `make --version` 可用 (GNU Make 4.3)
- [x] 所有检查通过

**调试日志**:
- debug1: 系统依赖检查全部通过, 执行所有 Tasks 验证, 猜想: 环境已预装所有依赖, 验证: 执行命令检查, 修复: 无需修复, 已修复

---

### Job 2: Go 安装

**目标**: 下载并安装 Go 1.21+

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 检查现有 Go 版本 (`go version`)
- [ ] Task 2: 如已安装且版本 >= 1.21，跳过安装
- [ ] Task 3: 确定下载 URL（根据 OS 和架构) 如果 网络不可用 则使用代理后尝试: export http_proxy=http://10.229.18.27:8412 && export https_proxy=http://10.229.18.27:8412
  - Linux x86_64: https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
  - Linux arm64: https://go.dev/dl/go1.21.0.linux-arm64.tar.gz
  - macOS x86_64: https://go.dev/dl/go1.21.0.darwin-amd64.tar.gz
  - macOS arm64: https://go.dev/dl/go1.21.0.darwin-arm64.tar.gz
- [ ] Task 4: 下载 Go 安装包到 `/tmp/`
- [ ] Task 5: 删除旧版本（如存在）`rm -rf /usr/local/go`
- [ ] Task 6: 解压到 `/usr/local/go`
- [ ] Task 7: 验证安装 `/usr/local/go/bin/go version`

**验证器**:
- [ ] 下载成功（文件大小 > 50MB）
- [ ] 解压后 `/usr/local/go/bin/go` 存在
- [ ] `go version` 显示 go1.21+ 或更高
- [ ] 无权限错误

**调试日志**:
- 待填充

---

### Job 3: Go 环境配置

**目标**: 配置 GOPATH 和 PATH

**前置条件**:
- Job 2 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 GOPATH 目录 `$HOME/go`
- [ ] Task 2: 创建子目录 `$HOME/go/bin`, `$HOME/go/pkg`, `$HOME/go/src`
- [ ] Task 3: 检测用户 shell 类型 (bash/zsh)
- [ ] Task 4: 配置 PATH 包含 `/usr/local/go/bin`
  - bash: 修改 `~/.bashrc`
  - zsh: 修改 `~/.zshrc`
- [ ] Task 5: 配置 GOPATH 环境变量
  - `export GOPATH=$HOME/go`
  - `export PATH=$PATH:$GOPATH/bin`
- [ ] Task 6: 配置 GOPROXY（中国大陆可选）
  - `export GOPROXY=https://goproxy.cn,direct`
- [ ] Task 7: 加载配置 `source ~/.bashrc` 或 `~/.zshrc`

**验证器**:
- [ ] `go version` 在任意目录可用
- [ ] `go env GOPATH` 返回 `$HOME/go`
- [ ] `go env GOROOT` 返回 `/usr/local/go`
- [ ] `echo $PATH` 包含 go 和 GOPATH/bin
- [ ] 配置持久化（重启后仍有效）

**调试日志**:
- 待填充

---

### Job 4: Go 工具安装

**目标**: 安装必要的 Go 工具

**前置条件**:
- Job 3 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 安装 `gofmt`（随 Go 安装，验证可用）
- [ ] Task 2: 安装 `goimports`（可选，代码格式化）
  - `go install golang.org/x/tools/cmd/goimports@latest`
- [ ] Task 3: 安装 `golint`（可选，代码检查）
  - `go install golang.org/x/lint/golint@latest`
- [ ] Task 4: 安装 `staticcheck`（可选，静态分析）
  - `go install honnef.co/go/tools/cmd/staticcheck@latest`
- [ ] Task 5: 验证所有工具在 PATH 中

**验证器**:
- [ ] `which gofmt` 返回路径
- [ ] `which goimports` 返回路径（如安装）
- [ ] `goimports --help` 正常输出
- [ ] 工具可以从命令行直接调用

**调试日志**:
- 待填充

---

### Job 5: 开发环境验证

**目标**: 验证完整的 Go 开发环境

**前置条件**:
- Job 4 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建测试项目目录 `$HOME/go/src/hello`
- [ ] Task 2: 创建 `main.go` 文件
  ```go
  package main
  import "fmt"
  func main() {
      fmt.Println("Hello, Go!")
  }
  ```
- [ ] Task 3: 初始化模块 `go mod init hello`
- [ ] Task 4: 构建项目 `go build`
- [ ] Task 5: 运行程序 `./hello`
- [ ] Task 6: 运行测试 `go test ./...`（空测试通过）
- [ ] Task 7: 格式化代码 `gofmt -w main.go`
- [ ] Task 8: 清理测试项目

**验证器**:
- [ ] `go mod init` 成功创建 go.mod
- [ ] `go build` 成功生成可执行文件
- [ ] `./hello` 输出 "Hello, Go!"
- [ ] `go test` 通过
- [ ] `gofmt` 正常工作
- [ ] 无错误信息

**调试日志**:
- 待填充

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- [ ] 完整的安装流程: 依赖检查 → Go 安装 → 环境配置 → 工具安装 → 验证
- [ ] 重启后环境仍然有效
- [ ] 可以从任意目录使用 `go` 命令
- [ ] 可以构建、运行、测试 Go 项目
- [ ] 集成测试通过

**调试日志**:
- 待填充

---

## 环境变量配置示例

### ~/.bashrc 或 ~/.zshrc

```bash
# Go 环境配置
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

# 中国大陆用户建议设置
export GOPROXY=https://goproxy.cn,direct
export GO111MODULE=on
```

---

## 验证命令清单

```bash
# 验证安装
go version                    # 应显示 go1.21+
go env GOROOT                 # 应显示 /usr/local/go
go env GOPATH                 # 应显示 /home/user/go
go env GO111MODULE            # 应显示 on

# 验证工具
which gofmt                   # 应返回路径
which goimports               # 应返回路径（如安装）

# 验证构建
go build ./...                # 应成功构建
go test ./...                 # 应通过测试
go fmt ./...                  # 应格式化代码
```

---

## 常见问题

### 1. 权限不足
**解决**: 使用 `sudo` 安装到 `/usr/local/go`，或安装到 `$HOME/go`

### 2. 下载速度慢
**解决**: 使用国内镜像
```bash
wget https://studygolang.com/dl/golang/go1.21.0.linux-amd64.tar.gz
```

### 3. GOPATH 未生效
**解决**: 重新加载配置
```bash
source ~/.bashrc  # 或 ~/.zshrc
```

---

## 文件清单

- 系统依赖: wget/curl, tar, git, make
- Go 安装: /usr/local/go
- 环境配置: ~/.bashrc 或 ~/.zshrc
- Go 工作区: $HOME/go

---

**前置条件**: 此 Plan 必须在所有 Go 模块开发之前完成
