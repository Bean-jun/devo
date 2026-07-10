package tools

import "context"

type sessionIDKeyType struct{}

var sessionIDKey sessionIDKeyType

// WithSessionID 将 sessionID 注入 context，供工具在执行时读取。
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// SessionIDFromContext 从 context 中提取 sessionID，不存在时返回空字符串。
func SessionIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(sessionIDKey).(string)
	return v
}
