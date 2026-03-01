# Plan: 用户认证模块

## 模块概述

**模块职责**: 实现用户注册、登录、权限验证和会话管理功能

**对应 Research**:
- `.morty/research/auth-strategy.md` - 认证策略研究
- `.morty/research/security-best-practices.md` - 安全最佳实践

**现有实现参考**:
- `internal/auth/session.go` - 现有会话管理实现
- `pkg/crypto/hash.go` - 密码哈希工具

**依赖模块**: 无

**被依赖模块**: user_profile, api_gateway

## 接口定义

### 输入接口
- **Register**: 接收用户名、邮箱、密码，返回用户 ID
- **Login**: 接收用户名/邮箱和密码，返回会话 Token
- **Verify**: 接收 Token，返回用户信息和权限

### 输出接口
- **UserID**: 字符串，唯一用户标识
- **SessionToken**: JWT Token，包含用户 ID 和权限
- **UserInfo**: 用户基本信息（ID、用户名、邮箱、角色）

## 数据模型

```go
type User struct {
    ID           string
    Username     string
    Email        string
    PasswordHash string
    Role         string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Session struct {
    Token     string
    UserID    string
    ExpiresAt time.Time
}
```

## Jobs

---

### Job 1: 用户注册功能

#### 目标

实现用户注册接口，包括输入验证、密码哈希和用户创建

#### 前置条件

无

#### Tasks

- [ ] Task 1: 创建 User 数据模型和数据库表
- [ ] Task 2: 实现密码哈希和验证函数
- [ ] Task 3: 实现用户名和邮箱唯一性检查
- [ ] Task 4: 实现注册 API 端点
- [ ] Task 5: 编写单元测试覆盖所有边界情况

#### 验证器

- 用户名长度 3-20 字符，只允许字母数字下划线
- 邮箱格式符合 RFC 5322 标准
- 密码长度至少 8 字符，包含大小写字母和数字
- 重复用户名或邮箱返回明确错误信息
- 密码使用 bcrypt 哈希，cost factor >= 10
- 注册成功返回用户 ID，不返回密码哈希
- 数据库事务确保原子性

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 2: 用户登录功能

#### 目标

实现用户登录接口，验证凭证并生成会话 Token

#### 前置条件

- job_1 - 用户注册功能完成

#### Tasks

- [ ] Task 1: 实现用户凭证验证逻辑
- [ ] Task 2: 集成 JWT Token 生成
- [ ] Task 3: 实现登录 API 端点
- [ ] Task 4: 添加登录失败次数限制
- [ ] Task 5: 编写单元测试和集成测试

#### 验证器

- 支持用户名或邮箱登录
- 密码验证使用恒定时间比较
- 登录成功返回 JWT Token，有效期 24 小时
- Token 包含 user_id 和 role claims
- 5 次登录失败后锁定账户 15 分钟
- 登录失败返回通用错误信息（防止用户枚举）
- Token 签名使用 HS256 算法

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 3: Token 验证中间件

#### 目标

实现 HTTP 中间件验证请求中的 JWT Token

#### 前置条件

- job_2 - 用户登录功能完成

#### Tasks

- [ ] Task 1: 实现 JWT Token 解析和验证
- [ ] Task 2: 创建 HTTP 中间件拦截请求
- [ ] Task 3: 实现权限检查逻辑
- [ ] Task 4: 添加 Token 刷新机制
- [ ] Task 5: 编写中间件测试

#### 验证器

- 验证 Token 签名有效性
- 检查 Token 过期时间
- 提取用户信息并注入请求上下文
- 支持基于角色的访问控制（RBAC）
- Token 过期前 1 小时允许刷新
- 无效 Token 返回 401 Unauthorized
- 权限不足返回 403 Forbidden

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

## 集成测试

**触发条件**: 模块内所有 Jobs 完成

**验证器**:
- 完整的注册-登录-验证流程正常工作
- 并发注册不会产生重复用户
- Token 验证中间件正确拦截未授权请求
- 密码修改后旧 Token 失效
- 性能测试：登录 QPS >= 1000
- 安全测试：SQL 注入、XSS、CSRF 防护有效
