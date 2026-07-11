package logging

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type traceIDKeyType struct{}

var traceIDKey traceIDKeyType

type sessionIDKeyType struct{}

var sessionIDKey sessionIDKeyType

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, traceIDKey, traceID)
}

func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

func SessionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(sessionIDKey).(string)
	return v
}

func GenerateTraceID() string {
	return fmt.Sprintf("trace-%d-%d", time.Now().UnixNano(), rng.Int63())
}
