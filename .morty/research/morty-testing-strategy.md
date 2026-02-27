# Morty 生产级测试策略

**文档版本**: 1.0  
**创建日期**: 2026-02-27  
**目标**: 确保 Morty 项目生产环境可用性

---

## 执行摘要

为确保 Morty 项目达到生产级质量，需要建立**多层次、全覆盖**的测试体系：

- **单元测试** (60%): 85%+ 代码覆盖率
- **集成测试** (30%): 核心模块协作验证  
- **E2E测试** (10%): 关键用户场景验证
- **性能测试**: 验证生产环境性能指标
- **兼容性测试**: 跨平台、跨版本验证

**预计实施周期**: 4-5周  
**关键里程碑**: Phase 1-2 完成后即可达到基本生产可用

---

## 1. 现有测试覆盖分析

### 1.1 测试资产盘点

| 测试类型 | 文件数 | 覆盖模块 | 质量 | 缺口 |
|---------|--------|---------|------|------|
| 单元测试 | 58个 | 全部核心模块 | ✅ 良好 | 边界场景不足 |
| 集成测试 | 少量 | executor, callcli | ⚠️ 不足 | 缺少端到端集成 |
| Shell测试 | 3个 | git, logging, cli | ⚠️ 基础 | 场景覆盖少 |
| E2E测试 | 0个 | 无 | ❌ 缺失 | 完全缺失 |

### 1.2 关键测试缺口

#### ❌ 高优先级缺失:
1. **完整工作流测试** - Research → Plan → Doing 全流程
2. **状态一致性测试** - 并发更新、崩溃恢复、文件损坏
3. **AI CLI 真实集成测试** - 当前只有 Mock 测试
4. **错误场景覆盖** - 网络超时、磁盘满、权限不足
5. **性能基准测试** - 大规模 Job、状态文件性能
6. **跨平台兼容性** - Linux/macOS/Windows 验证

---

## 2. 测试策略概览

### 2.1 测试金字塔

```
        ┌─────────────┐
        │   E2E 测试   │  10% - 关键业务流程
        │   (4个场景)   │      (手动+自动)
        ├─────────────┤
        │  集成测试    │  30% - 模块间协作
        │ (10+场景)    │      (自动化)
        ├─────────────┤
        │   单元测试   │  60% - 函数级别
        │  (85%覆盖)   │      (全自动)
        └─────────────┘
```

### 2.2 测试分层策略

| 层次 | 目标 | 工具 | 执行频率 |
|------|------|------|---------|
| 单元测试 | 函数正确性 | go test | 每次提交 |
| 集成测试 | 模块协作 | go test + testify | PR 合并前 |
| E2E测试 | 用户场景 | Shell脚本 + Mock CLI | 发布前 |
| 性能测试 | 性能指标 | go test -bench | 每周 |
| 兼容性测试 | 跨平台 | Docker + CI | 发布前 |

---

## 3. 单元测试策略 (60%)

### 3.1 目标覆盖率: 85%+

### 3.2 关键模块测试清单

#### A. 状态管理 (state/)

**已有测试** ✅:
- 状态转换验证
- 并发读写安全
- JSON 序列化

**缺失测试** ❌:
```go
// tests/unit/state/corruption_test.go
func TestStateFileCorruptionRecovery(t *testing.T)
func TestStateBackupRestore(t *testing.T)

// tests/unit/state/performance_test.go  
func TestLargeStateLoad1000Jobs(t *testing.T)
func TestLargeStateSave1000Jobs(t *testing.T)

// tests/unit/state/concurrent_test.go
func TestConcurrentStateUpdates(t *testing.T)
func TestStateRaceConditions(t *testing.T)
```

#### B. 执行引擎 (executor/)

**已有测试** ✅:
- Job 基本执行流程
- 状态转换

**缺失测试** ❌:
```go
// tests/unit/executor/retry_test.go
func TestRetryBoundary_ZeroRetries(t *testing.T)
func TestRetryBoundary_MaxRetries(t *testing.T)
func TestRetryBoundary_ExceedMax(t *testing.T)

// tests/unit/executor/timeout_test.go
func TestJobTimeout(t *testing.T)
func TestTaskTimeout(t *testing.T)

// tests/unit/executor/failure_test.go
func TestTaskFailurePropagation(t *testing.T)
func TestJobFailureHandling(t *testing.T)
```

#### C. 解析器 (parser/)

**已有测试** ✅:
- 标准 Plan 文档解析

**缺失测试** ❌:
```go
// tests/unit/parser/malformed_test.go
func TestMalformedMarkdown(t *testing.T)
func TestMissingRequiredFields(t *testing.T)
func TestInvalidJobFormat(t *testing.T)

// tests/unit/parser/unicode_test.go
func TestUnicodeInJobName(t *testing.T)
func TestChineseCharacters(t *testing.T)

// tests/unit/parser/large_file_test.go
func TestLargePlanParsing(t *testing.T) // >100KB
```

#### D. Git 集成 (git/)

**已有测试** ✅:
- 基本提交功能

**缺失测试** ❌:
```go
// tests/unit/git/conflict_test.go
func TestMergeConflictDetection(t *testing.T)
func TestConflictHandling(t *testing.T)

// tests/unit/git/degradation_test.go
func TestGitUnavailable(t *testing.T)
func TestGitPermissionDenied(t *testing.T)

// tests/unit/git/performance_test.go
func TestLargeCommitPerformance(t *testing.T)
```

### 3.3 实施计划

```bash
# Week 1: 补充状态管理测试
- 实现 corruption_test.go
- 实现 performance_test.go
- 实现 concurrent_test.go
- 目标: state/ 覆盖率 → 90%

# Week 2: 补充执行引擎测试  
- 实现 retry_test.go
- 实现 timeout_test.go
- 实现 failure_test.go
- 目标: executor/ 覆盖率 → 85%

# Week 3: 补充解析器和 Git 测试
- 实现 parser 边界测试
- 实现 git 边界测试
- 目标: 整体覆盖率 → 85%
```

---

## 4. 集成测试策略 (30%)

### 4.1 核心集成路径

#### 路径 1: DoingHandler → Executor → State

**测试场景**:
```go
// tests/integration/doing_flow_test.go

func TestCompleteJobExecutionFlow(t *testing.T) {
    // 1. Setup 测试环境
    tmpDir := setupTestEnvironment(t)
    defer cleanup(tmpDir)
    
    // 2. 创建 Mock AI CLI
    mockCLI := &MockAICLI{
        responses: map[string]string{
            "task1": "Task 1 completed",
            "task2": "Task 2 completed",
        },
    }
    
    // 3. 初始化组件
    stateManager := state.NewManager(tmpDir + "/.morty/status.json")
    executor := executor.NewEngineWithCLI(mockCLI, stateManager, ...)
    
    // 4. 执行 Job
    err := executor.ExecuteJob(ctx, "module1", "job1")
    assert.NoError(t, err)
    
    // 5. 验证状态持久化
    stateManager2 := state.NewManager(tmpDir + "/.morty/status.json")
    stateManager2.Load()
    status, _ := stateManager2.GetJobStatus("module1", "job1")
    assert.Equal(t, state.StatusCompleted, status)
}

func TestJobFailureAndRetry(t *testing.T) {
    // 测试失败重试流程
}

func TestJobExecutionWithStateCorruption(t *testing.T) {
    // 测试状态文件损坏时的处理
}
```

#### 路径 2: Executor → AI CLI → Git

**测试场景**:
```go
// tests/integration/execution_commit_test.go

func TestTaskExecutionWithGitCommit(t *testing.T) {
    // 验证 Task 执行后自动 Git 提交
}

func TestAICliTimeout(t *testing.T) {
    // 验证 AI CLI 超时处理
}

func TestGitCommitFailureHandling(t *testing.T) {
    // 验证 Git 提交失败时的回滚
}
```

#### 路径 3: 崩溃恢复

**测试场景**:
```go
// tests/integration/crash_recovery_test.go

func TestCrashDuringJobExecution(t *testing.T) {
    // 1. 启动 Job 执行
    // 2. 执行到一半模拟崩溃
    // 3. 重启验证状态正确
    // 4. 继续执行完成
}

func TestStateConsistencyAfterCrash(t *testing.T) {
    // 验证崩溃后状态一致性
}
```

### 4.2 并发测试

```go
// tests/integration/concurrent_test.go

func TestConcurrentJobExecution(t *testing.T) {
    // 并发执行多个独立 Job
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(jobID int) {
            defer wg.Done()
            executor.ExecuteJob(ctx, "module1", fmt.Sprintf("job%d", jobID))
        }(i)
    }
    wg.Wait()
    
    // 验证所有 Job 状态正确
}

func TestConcurrentStateUpdates(t *testing.T) {
    // 验证并发状态更新的正确性
}
```

---

## 5. E2E 测试策略 (10%)

### 5.1 核心场景

#### 场景 1: 新项目完整流程

```bash
#!/bin/bash
# tests/e2e/test_new_project.sh

set -e
TEST_DIR=$(mktemp -d)
cd $TEST_DIR
git init

# 1. Research
echo "=== Test Research ==="
morty research "implement auth" <<EOF
JWT authentication
EOF
test -f .morty/research/implement-auth.md
echo "✓ Research passed"

# 2. Plan  
echo "=== Test Plan ==="
morty plan
test -f .morty/plan/*.md
echo "✓ Plan passed"

# 3. Doing (使用 Mock CLI)
echo "=== Test Doing ==="
export CLAUDE_CODE_CLI="./mock_claude.sh"
morty doing --module auth --job setup
STATUS=$(morty stat | grep "auth/setup" | awk '{print $2}')
test "$STATUS" = "COMPLETED"
echo "✓ Doing passed"

# 4. Git 验证
COMMIT=$(git log -1 --pretty=%B)
echo "$COMMIT" | grep "morty: auth/setup"
echo "✓ Git commit passed"

rm -rf $TEST_DIR
echo "=== E2E Test PASSED ==="
```

#### 场景 2: 失败重试

```bash
#!/bin/bash
# tests/e2e/test_failure_retry.sh

# 1. 第一次执行失败
export MOCK_FAIL=true
morty doing && exit 1 || echo "Expected failure"

# 2. 验证状态为 FAILED
STATUS=$(morty stat | grep job1 | awk '{print $2}')
test "$STATUS" = "FAILED"

# 3. 修复后重试
export MOCK_FAIL=false
morty doing --restart

# 4. 验证成功
STATUS=$(morty stat | grep job1 | awk '{print $2}')
test "$STATUS" = "COMPLETED"
```

#### 场景 3: 状态恢复

```bash
#!/bin/bash
# tests/e2e/test_state_recovery.sh

# 1. 启动执行
morty doing &
PID=$!

# 2. 执行到一半 kill
sleep 5
kill -9 $PID

# 3. 验证状态为 RUNNING
STATUS=$(morty stat | grep job1 | awk '{print $2}')
test "$STATUS" = "RUNNING"

# 4. 重启继续执行
morty doing

# 5. 验证完成
STATUS=$(morty stat | grep job1 | awk '{print $2}')
test "$STATUS" = "COMPLETED"
```

#### 场景 4: 大规模项目

```bash
#!/bin/bash
# tests/e2e/test_large_scale.sh

# 生成 10 Modules x 20 Jobs = 200 Jobs
generate_large_plan 10 20

# 执行并监控
time morty doing

# 验证性能指标
- 总执行时间 < 10分钟 (Mock CLI)
- 内存占用 < 100MB
- 状态文件 < 10MB
```

### 5.2 Mock AI CLI 实现

```bash
#!/bin/bash
# tests/mocks/mock_claude.sh

# 模拟 Claude CLI 行为
sleep 0.1  # 模拟延迟

# 根据输入返回模拟输出
if [[ "$MOCK_FAIL" == "true" ]]; then
    echo "Error: Task failed"
    exit 1
fi

echo "Task completed successfully"
exit 0
```

---

## 6. 性能测试策略

### 6.1 性能指标

| 指标 | 目标值 | 测试方法 |
|------|--------|---------|
| 单个 Job 执行 | < 30s (不含 AI) | Benchmark |
| 状态文件读取 | < 100ms (1000 Jobs) | Benchmark |
| 状态文件写入 | < 200ms (1000 Jobs) | Benchmark |
| 内存占用 | < 100MB (100 Jobs) | 压力测试 |
| 并发执行 | 支持 10 并发 | 并发测试 |

### 6.2 性能测试实现

```go
// tests/performance/state_benchmark_test.go

func BenchmarkStateLoad1000Jobs(b *testing.B) {
    stateFile := generateStateFile(1000)
    manager := state.NewManager(stateFile)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        manager.Load()
    }
}

func BenchmarkStateSave1000Jobs(b *testing.B) {
    manager := setupWith1000Jobs()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        manager.Save()
    }
}

// tests/performance/executor_benchmark_test.go

func BenchmarkJobExecution(b *testing.B) {
    executor := setupMockExecutor()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        executor.ExecuteJob(ctx, "m1", "j1")
    }
}
```

### 6.3 压力测试

```bash
#!/bin/bash
# tests/stress/large_scale.sh

# 生成 1000 Jobs
generate_plan 1000

# 监控执行
start_monitoring  # CPU, Memory, I/O

# 执行
time morty doing

# 验证
check_no_memory_leak
check_no_deadlock
check_state_consistency
```

---

## 7. 兼容性测试策略

### 7.1 测试矩阵

| 维度 | 测试范围 |
|------|---------|
| **OS** | Ubuntu 20.04/22.04, CentOS 7/8, macOS 12-14, Windows 10/11 |
| **Go** | 1.21.x, 1.22.x, 1.23.x |
| **Git** | 2.25+, 2.30+, 2.40+ |

### 7.2 自动化测试

```yaml
# .github/workflows/compatibility.yml

name: Compatibility Tests

on: [push, pull_request]

jobs:
  cross-platform:
    strategy:
      matrix:
        os: [ubuntu-20.04, ubuntu-22.04, macos-12, macos-13, windows-2022]
        go: ['1.21', '1.22', '1.23']
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go }}
      - run: ./scripts/build.sh
      - run: make test-unit
      - run: make test-integration
```

---

## 8. Mock 策略

### 8.1 Mock AI CLI

```go
// tests/mocks/mock_ai_cli.go

type MockAICLI struct {
    Responses map[string]string
    CallCount int
    Latency   time.Duration
    FailCount int
}

func (m *MockAICLI) Call(ctx context.Context, prompt string) (string, error) {
    m.CallCount++
    time.Sleep(m.Latency)
    
    // 模拟失败
    if m.FailCount > 0 {
        m.FailCount--
        return "", fmt.Errorf("mock failure")
    }
    
    // 返回预定义响应
    for key, resp := range m.Responses {
        if strings.Contains(prompt, key) {
            return resp, nil
        }
    }
    
    return "default response", nil
}
```

### 8.2 Mock Git

```go
// tests/mocks/mock_git.go

type MockGit struct {
    Commits []string
    FailOnCommit bool
}

func (m *MockGit) CreateCommit(msg string) error {
    if m.FailOnCommit {
        return fmt.Errorf("commit failed")
    }
    m.Commits = append(m.Commits, msg)
    return nil
}
```

---

## 9. CI/CD 集成

### 9.1 测试流水线

```yaml
# .github/workflows/test.yml

name: Test Suite

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: make test-unit
      - run: make test-coverage
      - uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
  
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: make test-integration
  
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: make test-e2e
  
  performance-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      - run: make test-performance
```

### 9.2 质量门禁

```yaml
# PR 合并要求:
- ✅ 单元测试通过
- ✅ 集成测试通过  
- ✅ 代码覆盖率 ≥ 85%
- ✅ 无 lint 错误
- ✅ E2E 测试通过 (发布前)
```

---

## 10. 实施计划

### Phase 1: 基础测试完善 (Week 1-2)

**目标**: 达到 85% 单元测试覆盖率

- [ ] 补充状态管理边界测试
- [ ] 补充执行引擎边界测试
- [ ] 补充解析器容错测试
- [ ] 实现 Mock AI CLI 和 Git
- [ ] 搭建本地测试环境

**交付物**:
- 新增 15+ 单元测试文件
- Mock 框架实现
- 覆盖率报告 (85%+)

### Phase 2: 集成测试 (Week 3)

**目标**: 验证核心模块协作

- [ ] 实现完整 Job 执行流程测试
- [ ] 实现崩溃恢复测试
- [ ] 实现并发执行测试
- [ ] 实现状态一致性测试

**交付物**:
- 10+ 集成测试场景
- 集成测试框架

### Phase 3: E2E 测试 (Week 4)

**目标**: 验证用户关键场景

- [ ] 实现新项目完整流程测试
- [ ] 实现失败重试测试
- [ ] 实现状态恢复测试
- [ ] 实现大规模项目测试

**交付物**:
- 4 个 E2E 测试脚本
- Mock Claude CLI 实现
- E2E 测试文档

### Phase 4: 性能和兼容性 (Week 5)

**目标**: 验证生产环境性能

- [ ] 实现性能基准测试
- [ ] 实现压力测试
- [ ] 配置跨平台自动化测试
- [ ] 性能优化 (如发现瓶颈)

**交付物**:
- 性能基准报告
- 跨平台兼容性矩阵
- 性能优化建议

### Phase 5: CI/CD 集成 (Week 5)

**目标**: 自动化测试流水线

- [ ] 配置 GitHub Actions
- [ ] 集成覆盖率报告
- [ ] 配置质量门禁
- [ ] 编写测试文档

**交付物**:
- CI/CD 配置文件
- 自动化测试流水线
- 测试执行文档

---

## 11. 测试清单

### 11.1 开发阶段

```bash
# 每次提交前
✓ make test-unit        # 单元测试
✓ make lint             # 代码检查
✓ make coverage         # 覆盖率 ≥ 85%
```

### 11.2 PR 合并前

```bash
✓ 单元测试通过
✓ 集成测试通过
✓ 代码覆盖率 ≥ 85%
✓ 无 lint 错误
✓ 性能无回归
```

### 11.3 发布前

```bash
✓ 完整测试套件通过
✓ E2E 测试通过
✓ 跨平台兼容性验证
✓ 性能基准测试
✓ 安全扫描
✓ 手动测试关键场景
```

---

## 12. 成功标准

### 生产可用标准:

✅ **功能完整性**
- 所有核心功能测试通过
- 无已知 P0/P1 缺陷

✅ **质量标准**
- 单元测试覆盖率 ≥ 85%
- 集成测试覆盖核心流程
- E2E 测试覆盖主要场景

✅ **性能标准**
- Job 执行 < 30s (不含 AI)
- 状态操作 < 200ms (1000 Jobs)
- 内存 < 100MB (100 Jobs)

✅ **稳定性标准**
- 无内存泄漏
- 无竞态条件
- 崩溃恢复正常

✅ **兼容性标准**
- 支持主流 OS
- 支持 Go 1.21+
- 支持 Git 2.25+

---

## 13. 结论

通过实施上述全面的测试策略，Morty 项目可以达到**生产级质量标准**。

**关键要素**:
1. 高覆盖率单元测试 (85%+)
2. 完整集成测试 (核心流程)
3. 关键场景 E2E 测试
4. 持续性能和兼容性验证
5. 自动化 CI/CD 流程

**预计时间**: 4-5周  
**最小可用**: Phase 1-2 完成 (2-3周)

**投资回报**:
- 减少生产环境缺陷 80%+
- 提升代码质量和可维护性
- 加速后续功能开发
- 增强用户信心

