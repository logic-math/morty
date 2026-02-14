# Loop 和 Obs 功能合并总结

## 概述

将 `morty loop` 和 `morty obs` 功能合并,当用户启动 loop 时自动在 tmux 中启动三面板监控,循环在后台运行不受终端关闭影响。

## 主要变更

### 1. morty_loop.sh 重构

**新增功能:**
- 添加 `--no-monitor` 参数(默认启动 tmux 监控)
- 添加 `check_project_structure()` 函数(复用逻辑)
- 监控模式检测和 tmux 安装检查
- 委托给 `lib/loop_monitor.sh` 处理 tmux 会话

**新增参数:**
```bash
morty loop                    # 默认启动 tmux 监控(推荐)
morty loop --no-monitor       # 不启动监控,直接运行
morty loop --max-loops 100    # 自定义最大循环次数
```

### 2. lib/loop_monitor.sh (新文件)

**功能:**
- 创建 tmux 会话并配置三面板布局
- 生成后台循环执行脚本(loop_runner.sh)
- 配置三个面板:
  1. 左侧(60%): Loop 循环执行
  2. 右上: 实时日志尾随
  3. 右下: 状态监控 + 交互式 bash

**特性:**
- 循环在独立脚本中运行,不受 tmux 面板关闭影响
- 状态监控每 3 秒刷新一次
- 提供便捷命令: `status`, `logs`, `plan`
- 自动聚焦到交互式 bash 面板

### 3. 移除 morty_obs.sh

**原因:**
- obs 功能已集成到 loop 中
- 不再需要单独的 obs 命令
- 简化用户体验(一个命令完成所有功能)

### 4. 更新主命令和文档

**morty 命令:**
- 移除 `obs` 子命令
- 更新帮助文档反映集成监控

**install.sh:**
- 移除 morty_obs.sh 复制逻辑
- 添加 lib 脚本权限设置
- 更新快速开始指南

**README.md:**
- 更新命令文档
- 添加 tmux 快捷键说明
- 更新工作流程示例
- 更新版本号到 0.3.0

## 用户体验改进

### 之前的工作流程
```bash
# 步骤 1: 启动 obs
morty obs

# 步骤 2: 在左侧面板手动输入
morty loop
```

**问题:**
- 需要两个命令
- 需要手动在 tmux 面板中输入命令
- 容易忘记启动 obs

### 现在的工作流程
```bash
# 一个命令完成所有功能
morty loop
```

**优势:**
- 一个命令自动启动监控
- 自动配置所有面板
- 循环在后台运行
- 可以随时分离会话(Ctrl+B D)

## 技术实现细节

### 后台循环执行

循环逻辑被提取到独立的 `loop_runner.sh` 脚本中,该脚本:
- 在 tmux 左侧面板中执行
- 不依赖 tmux 会话存活
- 包含完整的循环逻辑(从 morty_loop.sh 复制)
- 输出直接显示在面板中

### 三面板布局

```
┌─────────────────┬──────────────┐
│                 │  实时日志    │
│  Loop 执行      ├──────────────┤
│  (60%)          │  状态 + Bash │
└─────────────────┴──────────────┘
```

**面板 0(左侧):**
- 执行 loop_runner.sh
- 显示循环执行过程
- 60% 宽度

**面板 1(右上):**
- tail -f 实时日志
- 显示 Claude Code 输出
- 自动跟随最新日志

**面板 2(右下):**
- 状态监控(每 3 秒刷新)
- 交互式 bash 终端
- 便捷命令: status, logs, plan

### tmux 会话管理

**会话命名:**
- 格式: `morty-loop-<timestamp>`
- 示例: `morty-loop-1707900000`

**重新连接:**
```bash
# 查看所有会话
tmux list-sessions

# 重新连接
tmux attach -t morty-loop-1707900000
```

**分离会话:**
```bash
# 在 tmux 中按下
Ctrl+B 然后 D
```

## 兼容性

### 向后兼容
- `--no-monitor` 参数保留了原有的直接运行模式
- 所有原有的 loop 参数仍然有效
- 不影响现有的 .morty/ 项目结构

### 依赖要求
- **必需**: tmux (用于监控模式)
- **可选**: jq (用于 JSON 解析,状态监控更美观)

如果 tmux 未安装,脚本会提示安装方法并建议使用 `--no-monitor` 参数。

## 测试建议

### 基本测试
```bash
# 1. 创建测试 PRD
cat > test_prd.md << 'EOF'
# 测试项目
简单的测试项目用于验证 Morty 功能。
EOF

# 2. 运行 fix 模式
morty fix test_prd.md

# 3. 启动带监控的循环
morty loop

# 4. 验证三个面板都正常工作
# 5. 测试分离会话(Ctrl+B D)
# 6. 重新连接会话
tmux attach -t <session-name>
```

### 边界情况测试
```bash
# 1. 无 tmux 环境
morty loop --no-monitor

# 2. 自定义参数
morty loop --max-loops 10 --delay 3

# 3. 中断循环(Ctrl+C)
# 4. 关闭 tmux 面板后循环是否继续
```

## 文件清单

### 修改的文件
- `morty_loop.sh` - 添加监控模式支持
- `morty` - 移除 obs 命令
- `install.sh` - 更新安装逻辑
- `README.md` - 更新文档

### 新增的文件
- `lib/loop_monitor.sh` - tmux 监控集成

### 删除的文件
- `morty_obs.sh` - 功能已集成到 loop

## Git 提交

```bash
commit 0ea6f86
feat(loop): merge loop and obs functionality with integrated tmux monitoring

将 loop 和 obs 功能合并,当启动 loop 时自动在 tmux 中启动三面板监控。
```

## 后续改进建议

1. **配置文件支持**
   - 允许用户自定义面板布局
   - 配置默认参数

2. **会话恢复**
   - 检测已存在的 morty-loop 会话
   - 提示用户是否重新连接

3. **日志过滤**
   - 在右上面板添加日志过滤选项
   - 只显示错误或警告

4. **状态持久化**
   - 保存会话信息到 .morty/.tmux_session
   - 提供 morty attach 命令快速重连

5. **性能优化**
   - 优化日志尾随性能
   - 减少状态刷新频率

## 总结

这次重构成功地将 loop 和 obs 功能合并,提供了更好的用户体验:
- ✅ 一个命令启动所有功能
- ✅ 自动配置 tmux 监控
- ✅ 后台运行支持
- ✅ 向后兼容
- ✅ 文档完善

用户现在可以用一个 `morty loop` 命令获得完整的开发循环体验,无需额外的配置或命令。
