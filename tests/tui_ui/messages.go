package main

import "time"

// ─── 消息模型 ───

type Role int

const (
	RoleUser Role = iota
	RoleAssistant
	RoleSystem
)

func (r Role) String() string {
	switch r {
	case RoleUser:
		return "user"
	case RoleAssistant:
		return "assistant"
	case RoleSystem:
		return "system"
	}
	return "unknown"
}

type Message struct {
	Role      Role
	Content   string
	Thinking  string
	ToolCalls []ToolCall
	Time      string
}

type ToolCall struct {
	Name     string
	Summary  string
	Status   string // "executing" | "success" | "error" | "pending"
	Duration string
	Diff     string
	Expanded bool
}

// ─── 会话模型 ───

type Session struct {
	Name         string
	MsgCount     int
	LastMsg      string
	LastActivity string
	Active       bool
}

// ─── 丰富 Mock 数据 ───

func mockSessions() []Session {
	return []Session{
		{Name: "修复登录页面Bug", MsgCount: 12, LastMsg: "请帮我修复登录页面的空指针异常...", LastActivity: "2小时前", Active: false},
		{Name: "demo-session", MsgCount: 5, LastMsg: "再帮我加一个单元测试", LastActivity: "刚刚", Active: true},
		{Name: "重构 API 层", MsgCount: 34, LastMsg: "把所有 handler 改为依赖注入模式", LastActivity: "昨天", Active: false},
		{Name: "数据库迁移方案", MsgCount: 8, LastMsg: "对比 PostgreSQL 和 MySQL 的迁移成本", LastActivity: "3天前", Active: false},
	}
}

func mockMessages() []Message {
	now := time.Now()
	t := func(offsetMin int) string {
		return now.Add(time.Duration(-offsetMin) * time.Minute).Format("15:04")
	}

	return []Message{
		// 系统消息
		{
			Role:    RoleSystem,
			Content: "session 已创建 · 2026-08-07 14:30",
			Time:    t(30),
		},

		// 第1轮：用户请求修复
		{
			Role:    RoleUser,
			Content: "帮我修复 utils.go 中的空指针问题，用户反馈在调用 GetUser 时偶尔会 panic",
			Time:    t(29),
		},
		{
			Role:     RoleAssistant,
			Content:  "我来分析一下 utils.go 中的空指针问题。\n\n首先定位到 `GetUser` 函数，该函数在 user 参数为 nil 时没有做防御性检查，导致直接访问 `user.Name` 时触发 panic。\n\n已修复问题，在 `GetUser` 函数开头添加了 nil 检查，并在 `oldFunc` 中添加了类似的防御性代码。",
			Thinking: "需要先读取 utils.go 找到具体位置\n找到 GetUser 函数，检查 nil 检查缺失\n在 oldFunc() 中也添加类似的防御性编程\n修复后写回文件",
			ToolCalls: []ToolCall{
				{
					Name: "read_file", Summary: "utils.go",
					Status: "success", Duration: "0.3s",
					Diff: "utils.go · 156 lines · 找到 GetUser 函数 (L42-L58)",
				},
				{
					Name: "write_file", Summary: "utils.go",
					Status: "success", Duration: "1.2s",
					Diff: `+ if user == nil { return "", fmt.Errorf("user is nil") }
- func GetUser(user *User) string {
+ func GetUser(user *User) (string, error) {
-   return user.Name
+   if user == nil { return "", fmt.Errorf("user is nil") }
+   return user.Name, nil`,
				},
			},
			Time: t(28),
		},

		// 第2轮：中文长文本
		{
			Role:    RoleUser,
			Content: "请帮我写一个完整的用户认证系统，包括注册、登录、JWT token 刷新、密码重置等功能。要求支持手机号和邮箱两种注册方式，密码需要加密存储，token 过期时间可配置。",
			Time:    t(25),
		},
		{
			Role:     RoleAssistant,
			Content:  "好的，我来为你设计一个完整的用户认证系统。\n\n这个系统包含以下模块：\n1. **用户注册**：支持手机号和邮箱两种方式\n2. **用户登录**：JWT token 认证\n3. **Token 刷新**：refresh token 机制\n4. **密码重置**：邮箱验证码重置\n\n以下是核心代码实现：\n\n```go\npackage auth\n\nimport (\n    \"crypto/rand\"\n    \"encoding/hex\"\n    \"time\"\n    \"github.com/golang-jwt/jwt/v5\"\n    \"golang.org/x/crypto/bcrypt\"\n)\n\ntype AuthService struct {\n    jwtSecret     []byte\n    tokenExpiry   time.Duration\n    refreshExpiry time.Duration\n}\n\nfunc NewAuthService(secret string, tokenExp, refreshExp time.Duration) *AuthService {\n    return &AuthService{\n        jwtSecret:     []byte(secret),\n        tokenExpiry:   tokenExp,\n        refreshExpiry: refreshExp,\n    }\n}\n```\n\n所有函数都已添加完整的错误处理和日志记录。",
			Thinking: "分析用户需求：需要完整的认证系统\n确定技术栈：JWT + bcrypt + refresh token\n设计模块结构：注册/登录/刷新/重置\n考虑安全性：密码加密、token 过期、刷新令牌轮转\n编写核心代码实现",
			ToolCalls: []ToolCall{
				{
					Name: "write_file", Summary: "auth/service.go",
					Status: "success", Duration: "2.5s",
					Diff: `+ package auth
+ import (
+   "crypto/rand"
+   "github.com/golang-jwt/jwt/v5"
+   "golang.org/x/crypto/bcrypt"
+ )
+ type AuthService struct { ... }`,
				},
				{
					Name: "write_file", Summary: "auth/handler.go",
					Status: "success", Duration: "1.8s",
					Diff: `+ package auth
+ type RegisterRequest struct {
+   Username string + json:"username"
+   Password string + json:"password"
+   Email    string + json:"email"
+   Phone    string + json:"phone"
+ }`,
				},
				{
					Name: "write_file", Summary: "auth/middleware.go",
					Status: "success", Duration: "1.1s",
					Diff: `+ package auth
+ func AuthMiddleware(svc *AuthService) gin.HandlerFunc {
+   return func(c *gin.Context) { ... }
+ }`,
				},
			},
			Time: t(24),
		},

		// 第3轮：混合中英文
		{
			Role:    RoleUser,
			Content: "The test coverage for the auth module is too low. 请帮我补充单元测试，specifically for edge cases like invalid tokens, expired refresh tokens, and concurrent registration attempts.",
			Time:    t(20),
		},
		{
			Role:     RoleAssistant,
			Content:  "我来为 auth 模块补充单元测试，重点覆盖边界情况。\n\n测试覆盖以下场景：\n- ✅ 无效 token 被拒绝\n- ✅ 过期 refresh token 无法刷新\n- ✅ 并发注册不会产生重复用户\n- ✅ bcrypt 密码验证\n- ✅ 手机号和邮箱格式校验\n\n```go\nfunc TestInvalidToken(t *testing.T) {\n    svc := NewAuthService(\"test-secret\", time.Hour, 24*time.Hour)\n    _, err := svc.ValidateToken(\"invalid-token\")\n    assert.Error(t, err)\n}\n```",
			Thinking: "需要补充 auth 模块的单元测试\n重点关注边界情况：\n  - invalid tokens\n  - expired refresh tokens\n  - concurrent registration\n  - password hashing\n  - input validation\n使用 table-driven tests 风格\n添加 mock 和 fixture",
			ToolCalls: []ToolCall{
				{
					Name: "write_file", Summary: "auth/service_test.go",
					Status: "success", Duration: "3.2s",
					Diff: `+ func TestInvalidToken(t *testing.T) { ... }
+ func TestExpiredRefreshToken(t *testing.T) { ... }
+ func TestConcurrentRegistration(t *testing.T) { ... }
+ func TestPasswordHashing(t *testing.T) { ... }`,
				},
				{
					Name: "execute_command", Summary: "go test -v -cover ./auth/",
					Status: "success", Duration: "4.5s",
					Diff: `=== RUN   TestInvalidToken --- PASS
=== RUN   TestExpiredRefreshToken --- PASS
=== RUN   TestConcurrentRegistration --- PASS
=== RUN   TestPasswordHashing --- PASS
PASS
coverage: 87.3% of statements
ok      devo/pkg/auth   4.523s`,
				},
			},
			Time: t(19),
		},

		// 第4轮：错误场景展示
		{
			Role:    RoleUser,
			Content: "部署到生产环境后报错了，日志显示数据库连接超时",
			Time:    t(10),
		},
		{
			Role:     RoleAssistant,
			Content:  "我来排查数据库连接超时的问题。\n\n先检查数据库配置和连接池设置。",
			Thinking: "生产环境数据库连接超时\n需要检查：\n  - 数据库连接字符串\n  - 连接池配置\n  - 网络连通性\n  - 慢查询日志",
			ToolCalls: []ToolCall{
				{
					Name: "read_file", Summary: "config/database.yaml",
					Status: "success", Duration: "0.2s",
					Diff: "host: db-prod.internal\nport: 5432\nmax_connections: 10\nconnect_timeout: 5s",
				},
				{
					Name: "execute_command", Summary: "ping db-prod.internal",
					Status: "error", Duration: "5.1s",
					Diff: "ping: cannot resolve db-prod.internal: Name or service not known",
				},
				{
					Name: "write_file", Summary: "config/database.yaml",
					Status: "pending", Duration: "",
					Diff: "- host: db-prod.internal\n+ host: 10.0.1.50\n+ connect_timeout: 30s\n+ max_connections: 20",
				},
			},
			Time: t(9),
		},
	}
}
