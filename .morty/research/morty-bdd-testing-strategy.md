# Morty BDD 测试策略 - 基于用户旅程

**文档版本**: 1.0  
**创建日期**: 2026-02-27  
**测试理念**: 基于用户行为和真实使用场景，通过 Shell 脚本模拟完整用户旅程

---

## 核心理念

### 测试原则

```
❌ 不写单元测试
❌ 不写集成测试  
❌ 不写传统E2E测试

✅ 只写 BDD 用户旅程测试
✅ 真实环境执行 morty 命令
✅ Mock AI CLI 隔离外部依赖
✅ 验证用户可观察的行为
```

### 测试环境

```
测试目录结构:

morty/                          # Morty 项目源码
├── bin/morty                   # 编译后的二进制
├── tests/
│   └── bdd/                    # BDD 测试套件
│       ├── mock_claude.sh      # Mock AI CLI ⭐
│       ├── test_helpers.sh     # 测试辅助函数
│       ├── scenarios/          # 用户场景测试
│       │   ├── 01_first_time_user.sh
│       │   ├── 02_daily_workflow.sh
│       │   ├── 03_error_recovery.sh
│       │   ├── 04_team_collaboration.sh
│       │   └── 05_large_project.sh
│       └── run_all.sh          # 测试运行器
│
/tmp/morty-test-*/              # 临时测试项目 ⭐
    ├── .git/                   # Git 仓库
    ├── .morty/                 # Morty 工作目录
    │   ├── status.json
    │   ├── research/
    │   ├── plan/
    │   └── doing/
    └── src/                    # 测试项目源码
```

---

## 1. Mock AI CLI 实现

### 1.1 Mock Claude CLI 核心

```bash
#!/bin/bash
# tests/bdd/mock_claude.sh
# 模拟 Claude Code CLI 的行为

set -e

MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/tmp/mock_claude_$(date +%s).log"

# 配置
MOCK_LATENCY=${MOCK_LATENCY:-0.5}  # 模拟延迟（秒）
MOCK_FAIL_RATE=${MOCK_FAIL_RATE:-0}  # 失败率 0-100
MOCK_RESPONSE_MODE=${MOCK_RESPONSE_MODE:-"auto"}  # auto|file|echo

# 日志函数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$LOG_FILE"
}

# 模拟延迟
simulate_latency() {
    sleep "$MOCK_LATENCY"
}

# 模拟失败
should_fail() {
    if [ "$MOCK_FAIL_RATE" -eq 0 ]; then
        return 1
    fi
    local rand=$((RANDOM % 100))
    [ "$rand" -lt "$MOCK_FAIL_RATE" ]
}

# 生成响应
generate_response() {
    local input="$1"
    
    case "$MOCK_RESPONSE_MODE" in
        "echo")
            # 简单回显模式
            echo "Task completed: $input"
            ;;
        "file")
            # 从文件读取响应
            local response_file="${MOCK_DIR}/responses/$(echo "$input" | md5sum | cut -d' ' -f1).txt"
            if [ -f "$response_file" ]; then
                cat "$response_file"
            else
                echo "Default response for: $input"
            fi
            ;;
        "auto")
            # 智能响应模式
            if echo "$input" | grep -qi "research"; then
                cat << 'EOF'
# Research Completed

## Summary
Research task completed successfully.

## Findings
1. Key insight 1
2. Key insight 2
3. Key insight 3

## Next Steps
- Proceed with planning phase
EOF
            elif echo "$input" | grep -qi "plan"; then
                cat << 'EOF'
# Development Plan

## Module: feature_implementation

### Job 1: setup
**Description**: Setup project structure
**Tasks**:
- Create directory structure
- Initialize configuration
- Setup dependencies

### Job 2: implementation
**Description**: Implement core functionality
**Tasks**:
- Write main logic
- Add error handling
- Implement tests

### Job 3: documentation
**Description**: Write documentation
**Tasks**:
- API documentation
- User guide
- Examples
EOF
            elif echo "$input" | grep -qi "task"; then
                cat << 'EOF'
✓ Task completed successfully

Changes made:
- Created new files
- Updated configuration
- Added tests

All checks passed.
EOF
            else
                echo "Task completed: $(echo "$input" | head -n 1)"
            fi
            ;;
    esac
}

# 主逻辑
main() {
    log "Mock Claude CLI called with args: $*"
    
    # 读取输入
    local input=""
    if [ -p /dev/stdin ]; then
        input=$(cat)
    fi
    
    log "Input: $input"
    
    # 模拟延迟
    simulate_latency
    
    # 检查是否应该失败
    if should_fail; then
        log "Simulating failure"
        echo "Error: Mock failure (MOCK_FAIL_RATE=$MOCK_FAIL_RATE%)" >&2
        exit 1
    fi
    
    # 生成响应
    local response=$(generate_response "$input")
    log "Response: $response"
    
    echo "$response"
    exit 0
}

main "$@"

### 1.2 Mock 配置文件

```bash
# tests/bdd/mock_config.sh
# Mock Claude CLI 配置

# 延迟配置
export MOCK_LATENCY=0.1  # 快速模式，0.1秒延迟

# 失败率配置
export MOCK_FAIL_RATE=0  # 默认不失败

# 响应模式
export MOCK_RESPONSE_MODE="auto"  # 自动智能响应

# 日志配置
export MOCK_LOG_ENABLED=true
export MOCK_LOG_DIR="/tmp/morty-mock-logs"

# Mock Claude CLI 路径
export CLAUDE_CODE_CLI="$(pwd)/tests/bdd/mock_claude.sh"
```

---

## 2. 测试辅助函数

```bash
#!/bin/bash
# tests/bdd/test_helpers.sh
# BDD 测试辅助函数库

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 测试计数器
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# 创建测试项目
create_test_project() {
    local project_name="${1:-test-project}"
    local test_dir="/tmp/morty-test-${project_name}-$$"
    
    echo -e "${BLUE}[SETUP]${NC} Creating test project: $test_dir"
    
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    # 初始化 Git
    git init -q
    git config user.email "test@morty.dev"
    git config user.name "Morty Test"
    
    # 创建基础项目结构
    mkdir -p src tests docs
    echo "# $project_name" > README.md
    git add .
    git commit -q -m "Initial commit"
    
    echo "$test_dir"
}

# 清理测试项目
cleanup_test_project() {
    local test_dir="$1"
    if [ -d "$test_dir" ]; then
        echo -e "${BLUE}[CLEANUP]${NC} Removing test project: $test_dir"
        rm -rf "$test_dir"
    fi
}

# 断言：命令成功
assert_success() {
    local cmd="$1"
    local description="$2"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    echo -e "${BLUE}[TEST]${NC} $description"
    echo -e "  ${YELLOW}Running:${NC} $cmd"
    
    if eval "$cmd" > /tmp/test_output_$$.log 2>&1; then
        echo -e "  ${GREEN}✓ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "  ${RED}✗ FAILED${NC}"
        echo -e "  ${RED}Output:${NC}"
        cat /tmp/test_output_$$.log | sed 's/^/    /'
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# 断言：命令失败
assert_failure() {
    local cmd="$1"
    local description="$2"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    echo -e "${BLUE}[TEST]${NC} $description"
    echo -e "  ${YELLOW}Running:${NC} $cmd"
    
    if eval "$cmd" > /tmp/test_output_$$.log 2>&1; then
        echo -e "  ${RED}✗ FAILED${NC} (expected failure but succeeded)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    else
        echo -e "  ${GREEN}✓ PASSED${NC} (failed as expected)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    fi
}

# 断言：文件存在
assert_file_exists() {
    local file="$1"
    local description="${2:-File should exist: $file}"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    echo -e "${BLUE}[TEST]${NC} $description"
    
    if [ -f "$file" ]; then
        echo -e "  ${GREEN}✓ PASSED${NC} File exists: $file"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "  ${RED}✗ FAILED${NC} File not found: $file"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# 断言：文件包含内容
assert_file_contains() {
    local file="$1"
    local pattern="$2"
    local description="${3:-File should contain: $pattern}"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    echo -e "${BLUE}[TEST]${NC} $description"
    
    if grep -q "$pattern" "$file" 2>/dev/null; then
        echo -e "  ${GREEN}✓ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "  ${RED}✗ FAILED${NC} Pattern not found in $file"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# 断言：Git 提交存在
assert_git_commit_exists() {
    local pattern="$1"
    local description="${2:-Git commit should exist with: $pattern}"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    echo -e "${BLUE}[TEST]${NC} $description"
    
    if git log --oneline | grep -q "$pattern"; then
        echo -e "  ${GREEN}✓ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "  ${RED}✗ FAILED${NC} Commit not found"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# 打印测试总结
print_test_summary() {
    echo ""
    echo "========================================"
    echo "           Test Summary"
    echo "========================================"
    echo "Total:  $TESTS_TOTAL"
    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo "========================================"
    
    if [ "$TESTS_FAILED" -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        return 1
    fi
}

# 等待函数
wait_for_file() {
    local file="$1"
    local timeout="${2:-10}"
    local elapsed=0
    
    while [ ! -f "$file" ] && [ $elapsed -lt $timeout ]; do
        sleep 0.5
        elapsed=$((elapsed + 1))
    done
    
    [ -f "$file" ]
}

---

## 3. 用户旅程场景

### 场景 1: 首次使用者 - 完整工作流

**用户故事**:
> 作为一个首次使用 Morty 的开发者，我想要从零开始创建一个新项目，经历完整的 Research → Plan → Doing 流程，最终看到代码变更被自动提交到 Git。

```bash
#!/bin/bash
# tests/bdd/scenarios/01_first_time_user.sh

set -e

# 加载辅助函数
source "$(dirname "$0")/../test_helpers.sh"
source "$(dirname "$0")/../mock_config.sh"

# Morty 二进制路径
MORTY_BIN="${MORTY_BIN:-$(pwd)/bin/morty}"

echo "========================================"
echo "  Scenario 1: First Time User Journey"
echo "========================================"
echo ""

# Given: 一个全新的项目
TEST_DIR=$(create_test_project "first-time-user")
cd "$TEST_DIR"

# When: 用户运行 morty research
echo ""
echo "Step 1: User runs 'morty research'"
echo "-----------------------------------"
assert_success \
    "echo 'I want to implement user authentication with JWT' | $MORTY_BIN research 'user-auth'" \
    "Research command should succeed"

assert_file_exists \
    ".morty/research/user-auth.md" \
    "Research document should be created"

assert_file_contains \
    ".morty/research/user-auth.md" \
    "authentication" \
    "Research document should contain relevant content"

# When: 用户运行 morty plan
echo ""
echo "Step 2: User runs 'morty plan'"
echo "-------------------------------"
assert_success \
    "$MORTY_BIN plan user-auth" \
    "Plan command should succeed"

assert_file_exists \
    ".morty/plan/user-auth.md" \
    "Plan document should be created"

assert_file_contains \
    ".morty/plan/user-auth.md" \
    "Job" \
    "Plan should contain Jobs"

# When: 用户运行 morty doing
echo ""
echo "Step 3: User runs 'morty doing'"
echo "--------------------------------"
assert_success \
    "$MORTY_BIN doing" \
    "Doing command should succeed"

# Then: 验证状态文件
assert_file_exists \
    ".morty/status.json" \
    "Status file should be created"

# Then: 验证 Git 提交
assert_git_commit_exists \
    "morty:" \
    "Git commit should exist with morty prefix"

# When: 用户查看状态
echo ""
echo "Step 4: User checks status"
echo "--------------------------"
assert_success \
    "$MORTY_BIN stat" \
    "Stat command should succeed"

# 清理
cleanup_test_project "$TEST_DIR"

# 总结
print_test_summary

### 场景 2: 日常开发工作流

**用户故事**:
> 作为一个日常使用 Morty 的开发者，我想要执行特定的 Module 和 Job，跳过不需要的步骤，快速完成开发任务。

```bash
#!/bin/bash
# tests/bdd/scenarios/02_daily_workflow.sh

set -e

source "$(dirname "$0")/../test_helpers.sh"
source "$(dirname "$0")/../mock_config.sh"

MORTY_BIN="${MORTY_BIN:-$(pwd)/bin/morty}"

echo "========================================"
echo "  Scenario 2: Daily Development Workflow"
echo "========================================"
echo ""

# Given: 一个已有 Plan 的项目
TEST_DIR=$(create_test_project "daily-workflow")
cd "$TEST_DIR"

# 准备：创建 Plan 文件
mkdir -p .morty/plan
cat > .morty/plan/feature.md << 'EOF'
# Feature Development Plan

## Module: api_endpoints

### Job 1: create_user_endpoint
**Description**: Create user creation endpoint
**Tasks**:
- Define API schema
- Implement handler
- Add validation

### Job 2: create_auth_endpoint
**Description**: Create authentication endpoint
**Tasks**:
- Implement login logic
- Generate JWT token
- Add rate limiting

## Module: database

### Job 1: setup_migrations
**Description**: Setup database migrations
**Tasks**:
- Create migration files
- Setup migration tool
- Test migrations
EOF

# When: 用户执行特定 Module 的 Job
echo ""
echo "Step 1: Execute specific module and job"
echo "----------------------------------------"
assert_success \
    "$MORTY_BIN doing --module api_endpoints --job create_user_endpoint" \
    "Should execute specific job"

# Then: 验证只有指定的 Job 被执行
assert_file_exists \
    ".morty/status.json" \
    "Status file should exist"

# When: 用户查看状态
echo ""
echo "Step 2: Check execution status"
echo "-------------------------------"
assert_success \
    "$MORTY_BIN stat" \
    "Should show current status"

# When: 用户继续执行下一个 Job
echo ""
echo "Step 3: Execute next job"
echo "------------------------"
assert_success \
    "$MORTY_BIN doing --module api_endpoints --job create_auth_endpoint" \
    "Should execute next job"

# Then: 验证 Git 提交
assert_git_commit_exists \
    "api_endpoints" \
    "Git commits should exist for executed jobs"

# 清理
cleanup_test_project "$TEST_DIR"

print_test_summary

### 场景 3: 错误恢复 - 失败重试

**用户故事**:
> 作为开发者，当 Job 执行失败时，我想要能够查看失败原因，修复问题后重新执行，最终成功完成任务。

```bash
#!/bin/bash
# tests/bdd/scenarios/03_error_recovery.sh

set -e

source "$(dirname "$0")/../test_helpers.sh"
source "$(dirname "$0")/../mock_config.sh"

MORTY_BIN="${MORTY_BIN:-$(pwd)/bin/morty}"

echo "========================================"
echo "  Scenario 3: Error Recovery Journey"
echo "========================================"
echo ""

# Given: 一个项目准备执行 Job
TEST_DIR=$(create_test_project "error-recovery")
cd "$TEST_DIR"

# 准备 Plan
mkdir -p .morty/plan
cat > .morty/plan/feature.md << 'EOF'
# Feature Plan

## Module: feature

### Job 1: risky_job
**Description**: A job that might fail
**Tasks**:
- Task 1
- Task 2
- Task 3
EOF

# When: 第一次执行失败（设置 Mock 失败）
echo ""
echo "Step 1: First execution fails"
echo "------------------------------"
export MOCK_FAIL_RATE=100  # 100% 失败率

assert_failure \
    "$MORTY_BIN doing --module feature --job risky_job" \
    "Job should fail on first attempt"

# Then: 验证状态为 FAILED
echo ""
echo "Step 2: Verify job status is FAILED"
echo "------------------------------------"
assert_success \
    "$MORTY_BIN stat | grep -i 'failed'" \
    "Status should show FAILED"

# When: 修复问题后重试
echo ""
echo "Step 3: Retry after fixing issue"
echo "---------------------------------"
export MOCK_FAIL_RATE=0  # 恢复正常

assert_success \
    "$MORTY_BIN doing --restart --module feature --job risky_job" \
    "Job should succeed after restart"

# Then: 验证状态为 COMPLETED
echo ""
echo "Step 4: Verify job status is COMPLETED"
echo "---------------------------------------"
assert_success \
    "$MORTY_BIN stat | grep -i 'completed'" \
    "Status should show COMPLETED"

# Then: 验证 Git 提交
assert_git_commit_exists \
    "morty: feature/risky_job" \
    "Git commit should exist after successful execution"

# 清理
cleanup_test_project "$TEST_DIR"

print_test_summary

### 场景 4: 团队协作 - 状态恢复

**用户故事**:
> 作为团队成员，当我拉取同事的代码后，我想要能够查看 Morty 的执行状态，继续未完成的工作，或者重置状态重新开始。

```bash
#!/bin/bash
# tests/bdd/scenarios/04_team_collaboration.sh

set -e

source "$(dirname "$0")/../test_helpers.sh"
source "$(dirname "$0")/../mock_config.sh"

MORTY_BIN="${MORTY_BIN:-$(pwd)/bin/morty}"

echo "========================================"
echo "  Scenario 4: Team Collaboration"
echo "========================================"
echo ""

# Given: 开发者 A 创建项目并执行部分工作
TEST_DIR=$(create_test_project "team-collab")
cd "$TEST_DIR"

echo ""
echo "Developer A: Initialize project"
echo "--------------------------------"

# 准备 Plan
mkdir -p .morty/plan
cat > .morty/plan/feature.md << 'EOF'
# Team Feature Plan

## Module: backend

### Job 1: api_setup
**Description**: Setup API structure
**Tasks**:
- Create routes
- Setup middleware

### Job 2: database_setup
**Description**: Setup database
**Tasks**:
- Create models
- Setup connections

## Module: frontend

### Job 1: ui_components
**Description**: Create UI components
**Tasks**:
- Create components
- Add styling
EOF

# A 执行第一个 Job
assert_success \
    "$MORTY_BIN doing --module backend --job api_setup" \
    "Developer A executes first job"

# A 提交代码
git add .morty/
git commit -m "Complete API setup"

# When: 开发者 B 克隆项目（模拟）
echo ""
echo "Developer B: Clone and check status"
echo "------------------------------------"

# B 查看状态
assert_success \
    "$MORTY_BIN stat" \
    "Developer B should see current status"

# B 继续执行下一个 Job
echo ""
echo "Developer B: Continue with next job"
echo "------------------------------------"
assert_success \
    "$MORTY_BIN doing --module backend --job database_setup" \
    "Developer B continues with next job"

# When: 开发者 C 想要重新开始
echo ""
echo "Developer C: Reset and restart"
echo "-------------------------------"

# C 查看历史
assert_success \
    "$MORTY_BIN reset -l 5" \
    "Should list recent commits"

# C 重置状态
assert_success \
    "$MORTY_BIN doing --restart" \
    "Should restart from beginning"

# 清理
cleanup_test_project "$TEST_DIR"

print_test_summary

### 场景 5: 大型项目 - 性能和稳定性

**用户故事**:
> 作为大型项目的开发者，我想要验证 Morty 在处理多个 Module 和大量 Job 时的性能和稳定性。

```bash
#!/bin/bash
# tests/bdd/scenarios/05_large_project.sh

set -e

source "$(dirname "$0")/../test_helpers.sh"
source "$(dirname "$0")/../mock_config.sh"

MORTY_BIN="${MORTY_BIN:-$(pwd)/bin/morty}"

echo "========================================"
echo "  Scenario 5: Large Project Performance"
echo "========================================"
echo ""

# Given: 一个大型项目
TEST_DIR=$(create_test_project "large-project")
cd "$TEST_DIR"

# 生成大型 Plan (5 Modules x 10 Jobs = 50 Jobs)
echo ""
echo "Step 1: Generate large plan"
echo "----------------------------"

mkdir -p .morty/plan
cat > .morty/plan/large-feature.md << 'EOF'
# Large Project Plan

## Module: auth
### Job 1: user_registration
**Tasks**: [Task 1, Task 2, Task 3]
### Job 2: user_login
**Tasks**: [Task 1, Task 2, Task 3]
### Job 3: password_reset
**Tasks**: [Task 1, Task 2, Task 3]
### Job 4: oauth_integration
**Tasks**: [Task 1, Task 2, Task 3]
### Job 5: session_management
**Tasks**: [Task 1, Task 2, Task 3]

## Module: api
### Job 1: rest_endpoints
**Tasks**: [Task 1, Task 2, Task 3]
### Job 2: graphql_setup
**Tasks**: [Task 1, Task 2, Task 3]
### Job 3: api_versioning
**Tasks**: [Task 1, Task 2, Task 3]
### Job 4: rate_limiting
**Tasks**: [Task 1, Task 2, Task 3]
### Job 5: api_docs
**Tasks**: [Task 1, Task 2, Task 3]

## Module: database
### Job 1: schema_design
**Tasks**: [Task 1, Task 2, Task 3]
### Job 2: migrations
**Tasks**: [Task 1, Task 2, Task 3]
### Job 3: indexes
**Tasks**: [Task 1, Task 2, Task 3]
### Job 4: backups
**Tasks**: [Task 1, Task 2, Task 3]
### Job 5: replication
**Tasks**: [Task 1, Task 2, Task 3]

## Module: frontend
### Job 1: ui_components
**Tasks**: [Task 1, Task 2, Task 3]
### Job 2: state_management
**Tasks**: [Task 1, Task 2, Task 3]
### Job 3: routing
**Tasks**: [Task 1, Task 2, Task 3]
### Job 4: forms
**Tasks**: [Task 1, Task 2, Task 3]
### Job 5: styling
**Tasks**: [Task 1, Task 2, Task 3]

## Module: testing
### Job 1: unit_tests
**Tasks**: [Task 1, Task 2, Task 3]
### Job 2: integration_tests
**Tasks**: [Task 1, Task 2, Task 3]
### Job 3: e2e_tests
**Tasks**: [Task 1, Task 2, Task 3]
### Job 4: performance_tests
**Tasks**: [Task 1, Task 2, Task 3]
### Job 5: security_tests
**Tasks**: [Task 1, Task 2, Task 3]
EOF

echo "Generated plan with 25 jobs across 5 modules"

# When: 执行所有 Jobs（使用快速 Mock）
echo ""
echo "Step 2: Execute all jobs (with performance monitoring)"
echo "------------------------------------------------------"

export MOCK_LATENCY=0.05  # 快速模式

# 记录开始时间
START_TIME=$(date +%s)

# 执行所有 Jobs
for module in auth api database frontend testing; do
    for job_num in {1..5}; do
        echo "Executing: $module/job_$job_num"
        $MORTY_BIN doing --module "$module" --job "job_$job_num" > /dev/null 2>&1 || true
    done
done

# 记录结束时间
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "Total execution time: ${DURATION}s"

# Then: 验证性能指标
echo ""
echo "Step 3: Verify performance metrics"
echo "-----------------------------------"

# 状态文件大小
STATUS_SIZE=$(stat -f%z .morty/status.json 2>/dev/null || stat -c%s .morty/status.json)
echo "Status file size: $STATUS_SIZE bytes"

if [ "$STATUS_SIZE" -lt 1048576 ]; then  # < 1MB
    echo -e "  ${GREEN}✓${NC} Status file size is acceptable"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "  ${RED}✗${NC} Status file too large"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
TESTS_TOTAL=$((TESTS_TOTAL + 1))

# 平均执行时间
AVG_TIME=$((DURATION / 25))
echo "Average time per job: ${AVG_TIME}s"

if [ "$AVG_TIME" -lt 5 ]; then
    echo -e "  ${GREEN}✓${NC} Performance is acceptable"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "  ${RED}✗${NC} Performance needs improvement"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
TESTS_TOTAL=$((TESTS_TOTAL + 1))

# Git 提交数量
COMMIT_COUNT=$(git log --oneline | grep "morty:" | wc -l)
echo "Git commits created: $COMMIT_COUNT"

# 清理
cleanup_test_project "$TEST_DIR"

print_test_summary

---

## 4. 测试运行器

```bash
#!/bin/bash
# tests/bdd/run_all.sh
# BDD 测试套件运行器

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MORTY_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "========================================"
echo "  Morty BDD Test Suite"
echo "========================================"
echo ""
echo "Morty Root: $MORTY_ROOT"
echo "Test Dir: $SCRIPT_DIR"
echo ""

# 1. 检查 Morty 二进制
echo "Step 1: Check Morty binary"
echo "---------------------------"
if [ ! -f "$MORTY_ROOT/bin/morty" ]; then
    echo -e "${RED}Error: Morty binary not found${NC}"
    echo "Please run: ./scripts/build.sh"
    exit 1
fi
echo -e "${GREEN}✓${NC} Morty binary found"

export MORTY_BIN="$MORTY_ROOT/bin/morty"

# 2. 检查 Mock CLI
echo ""
echo "Step 2: Check Mock CLI"
echo "----------------------"
if [ ! -f "$SCRIPT_DIR/mock_claude.sh" ]; then
    echo -e "${RED}Error: Mock Claude CLI not found${NC}"
    exit 1
fi
chmod +x "$SCRIPT_DIR/mock_claude.sh"
echo -e "${GREEN}✓${NC} Mock CLI ready"

# 3. 加载配置
source "$SCRIPT_DIR/mock_config.sh"

# 4. 运行场景测试
echo ""
echo "Step 3: Run scenario tests"
echo "--------------------------"
echo ""

SCENARIOS=(
    "01_first_time_user.sh"
    "02_daily_workflow.sh"
    "03_error_recovery.sh"
    "04_team_collaboration.sh"
    "05_large_project.sh"
)

TOTAL_SCENARIOS=${#SCENARIOS[@]}
PASSED_SCENARIOS=0
FAILED_SCENARIOS=0

for scenario in "${SCENARIOS[@]}"; do
    scenario_path="$SCRIPT_DIR/scenarios/$scenario"
    
    if [ ! -f "$scenario_path" ]; then
        echo -e "${RED}✗${NC} Scenario not found: $scenario"
        FAILED_SCENARIOS=$((FAILED_SCENARIOS + 1))
        continue
    fi
    
    echo ""
    echo "================================================"
    echo "Running: $scenario"
    echo "================================================"
    
    if bash "$scenario_path"; then
        echo -e "${GREEN}✓ PASSED${NC}: $scenario"
        PASSED_SCENARIOS=$((PASSED_SCENARIOS + 1))
    else
        echo -e "${RED}✗ FAILED${NC}: $scenario"
        FAILED_SCENARIOS=$((FAILED_SCENARIOS + 1))
    fi
done

# 5. 总结
echo ""
echo "========================================"
echo "         Final Summary"
echo "========================================"
echo "Total Scenarios: $TOTAL_SCENARIOS"
echo -e "Passed: ${GREEN}$PASSED_SCENARIOS${NC}"
echo -e "Failed: ${RED}$FAILED_SCENARIOS${NC}"
echo "========================================"
echo ""

if [ "$FAILED_SCENARIOS" -eq 0 ]; then
    echo -e "${GREEN}🎉 All scenarios passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some scenarios failed!${NC}"
    exit 1
fi

---

## 5. 实施步骤

### Phase 1: 搭建测试框架 (1-2天)

```bash
# 1. 创建测试目录结构
mkdir -p tests/bdd/{scenarios,responses}

# 2. 实现 Mock Claude CLI
cat > tests/bdd/mock_claude.sh << 'EOF'
[Mock CLI 代码见上文]
EOF
chmod +x tests/bdd/mock_claude.sh

# 3. 实现测试辅助函数
cat > tests/bdd/test_helpers.sh << 'EOF'
[测试辅助函数见上文]
EOF

# 4. 配置 Mock
cat > tests/bdd/mock_config.sh << 'EOF'
[Mock 配置见上文]
EOF

# 5. 测试 Mock CLI
export CLAUDE_CODE_CLI="./tests/bdd/mock_claude.sh"
echo "test input" | $CLAUDE_CODE_CLI
```

### Phase 2: 实现核心场景 (2-3天)

```bash
# 场景优先级
1. ✅ 场景 1: 首次使用者 (最重要)
2. ✅ 场景 3: 错误恢复 (高优先级)
3. ✅ 场景 2: 日常工作流
4. ✅ 场景 4: 团队协作
5. ✅ 场景 5: 大型项目
```

### Phase 3: 集成 CI/CD (1天)

```yaml
# .github/workflows/bdd-tests.yml
name: BDD Tests

on: [push, pull_request]

jobs:
  bdd-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build Morty
        run: ./scripts/build.sh
      
      - name: Run BDD Tests
        run: |
          cd tests/bdd
          chmod +x run_all.sh
          ./run_all.sh
      
      - name: Upload Test Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: bdd-test-results
          path: /tmp/morty-mock-logs/
```

---

## 6. 测试执行指南

### 6.1 本地运行单个场景

```bash
# 1. 构建 Morty
./scripts/build.sh

# 2. 运行单个场景
cd tests/bdd
chmod +x scenarios/01_first_time_user.sh
./scenarios/01_first_time_user.sh
```

### 6.2 本地运行所有场景

```bash
# 运行完整测试套件
cd tests/bdd
chmod +x run_all.sh
./run_all.sh
```

### 6.3 调试模式

```bash
# 启用详细日志
export MOCK_LOG_ENABLED=true
export MOCK_LOG_DIR="/tmp/morty-debug"

# 增加延迟观察执行过程
export MOCK_LATENCY=2.0

# 运行测试
./scenarios/01_first_time_user.sh
```

### 6.4 性能测试模式

```bash
# 快速模式（无延迟）
export MOCK_LATENCY=0

# 运行大型项目场景
./scenarios/05_large_project.sh
```

---

## 7. Mock 响应定制

### 7.1 预定义响应文件

```bash
# 为特定输入创建响应文件
mkdir -p tests/bdd/responses

# 创建响应
cat > tests/bdd/responses/user_auth_research.txt << 'EOF'
# User Authentication Research

## Overview
JWT-based authentication system with refresh tokens.

## Key Components
1. Authentication middleware
2. Token generation service
3. User session management

## Security Considerations
- Password hashing with bcrypt
- Token expiration
- Rate limiting
EOF

# 使用文件响应模式
export MOCK_RESPONSE_MODE="file"
```

### 7.2 动态响应逻辑

```bash
# 修改 mock_claude.sh 添加自定义逻辑
# 根据输入模式返回不同响应
if echo "$input" | grep -qi "authentication"; then
    # 返回认证相关响应
elif echo "$input" | grep -qi "database"; then
    # 返回数据库相关响应
fi
```

---

## 8. 验证清单

### 8.1 场景 1: 首次使用者
- [ ] Research 命令成功执行
- [ ] Research 文档生成
- [ ] Plan 命令成功执行
- [ ] Plan 文档包含 Jobs
- [ ] Doing 命令成功执行
- [ ] 状态文件正确生成
- [ ] Git 自动提交
- [ ] Stat 命令正常显示

### 8.2 场景 2: 日常工作流
- [ ] 可以指定 Module 执行
- [ ] 可以指定 Job 执行
- [ ] 状态正确跟踪
- [ ] 多次执行互不干扰

### 8.3 场景 3: 错误恢复
- [ ] 失败状态正确记录
- [ ] Restart 标志生效
- [ ] 重试后成功执行
- [ ] 状态正确更新

### 8.4 场景 4: 团队协作
- [ ] 状态可以跨会话恢复
- [ ] Reset 命令正常工作
- [ ] 历史查看功能正常

### 8.5 场景 5: 大型项目
- [ ] 支持多 Module
- [ ] 支持大量 Jobs (50+)
- [ ] 性能指标达标
- [ ] 状态文件大小合理
- [ ] 无内存泄漏

---

## 9. 成功标准

### 9.1 功能完整性
✅ 所有 5 个场景测试通过  
✅ 所有用户旅程验证通过  
✅ Mock CLI 稳定可靠

### 9.2 性能标准
✅ 单个 Job 执行 < 5s (Mock 模式)  
✅ 50 Jobs 执行 < 3分钟 (Mock 模式)  
✅ 状态文件 < 1MB (50 Jobs)

### 9.3 可维护性
✅ 测试代码清晰易读  
✅ 测试失败信息明确  
✅ 新场景易于添加

---

## 10. 与传统测试的对比

| 维度 | 传统测试 | BDD 用户旅程测试 |
|------|---------|-----------------|
| **测试粒度** | 函数/模块级别 | 用户场景级别 |
| **测试环境** | Mock 环境 | 真实环境 + Mock AI |
| **测试视角** | 开发者视角 | 用户视角 |
| **测试目标** | 代码正确性 | 用户体验 |
| **维护成本** | 高（代码变动影响大） | 低（关注行为不关注实现） |
| **业务价值** | 技术保障 | 直接验证业务价值 |
| **失败定位** | 精确到函数 | 定位到用户场景 |
| **执行速度** | 快（毫秒级） | 慢（秒级） |

---

## 11. 优势总结

### ✅ 为什么选择 BDD 用户旅程测试？

1. **真实性**
   - 在真实环境中执行 morty 命令
   - 验证真实的文件系统操作
   - 验证真实的 Git 操作

2. **可维护性**
   - 测试代码简洁明了
   - 不依赖内部实现细节
   - 重构代码不影响测试

3. **业务价值**
   - 直接验证用户能否完成任务
   - 覆盖端到端用户旅程
   - 发现真实使用中的问题

4. **快速反馈**
   - 5 个场景覆盖核心功能
   - 执行时间 < 5 分钟
   - 失败信息清晰直观

5. **易于扩展**
   - 新场景只需添加 Shell 脚本
   - 复用测试辅助函数
   - Mock 响应易于定制

---

## 12. 快速开始

```bash
# 1. 克隆项目
cd morty

# 2. 构建 Morty
./scripts/build.sh

# 3. 创建测试目录
mkdir -p tests/bdd/scenarios

# 4. 复制测试文件（从本文档）
# - mock_claude.sh
# - test_helpers.sh
# - mock_config.sh
# - scenarios/*.sh
# - run_all.sh

# 5. 运行测试
cd tests/bdd
./run_all.sh

# 6. 查看结果
# ✓ 绿色表示通过
# ✗ 红色表示失败
```

---

## 13. 总结

这套 BDD 测试策略：

✅ **专注用户价值** - 验证用户能否完成任务  
✅ **真实环境测试** - 在独立项目中执行真实命令  
✅ **Mock AI 隔离** - 隔离外部依赖，测试可控  
✅ **简洁易维护** - Shell 脚本清晰，易于理解和修改  
✅ **快速反馈** - 5 个场景 < 5 分钟完成  

**预计实施时间**: 3-4 天  
**维护成本**: 低  
**业务价值**: 高

---

**文档版本**: 1.0  
**创建日期**: 2026-02-27  
**状态**: ✅ 设计完成，待实施
