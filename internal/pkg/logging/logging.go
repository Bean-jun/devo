package logging

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *slog.Logger

type Config struct {
	Level      slog.Level
	LogPath    string
	WithCaller bool
}

func Init(cfg Config) {
	level := cfg.Level
	if level == 0 {
		level = slog.LevelInfo
	}

	var writer io.Writer
	if cfg.LogPath != "" {
		dir := filepath.Dir(cfg.LogPath)
		os.MkdirAll(dir, 0755)

		lj := &lumberjack.Logger{
			Filename:   cfg.LogPath,
			MaxSize:    50,
			MaxBackups: 10,
			MaxAge:     7,
			Compress:   true,
		}
		writer = io.MultiWriter(os.Stdout, lj)
	} else {
		writer = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	if cfg.WithCaller {
		opts.AddSource = true
	}

	handler := slog.NewTextHandler(writer, opts)
	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

func Debug(ctx context.Context, msg string, args ...any) {
	log(ctx, slog.LevelDebug, msg, args)
}

func Info(ctx context.Context, msg string, args ...any) {
	log(ctx, slog.LevelInfo, msg, args)
}

func Warn(ctx context.Context, msg string, args ...any) {
	log(ctx, slog.LevelWarn, msg, args)
}

func Error(ctx context.Context, msg string, args ...any) {
	log(ctx, slog.LevelError, msg, args)
}

func log(ctx context.Context, level slog.Level, msg string, args []any) {
	if Logger == nil {
		return
	}

	attrs := make([]slog.Attr, 0, len(args)/2+3)

	if traceID := TraceIDFromContext(ctx); traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}
	if sessionID := SessionIDFromContext(ctx); sessionID != "" {
		attrs = append(attrs, slog.String("session_id", sessionID))
	}

	if len(args)%2 != 0 {
		args = append(args, "!MISSING_VALUE")
	}
	for i := 0; i < len(args); i += 2 {
		key, _ := args[i].(string)
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}

	if !Logger.Enabled(context.Background(), level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = Logger.Handler().Handle(ctx, r)
}

func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Request-ID")
		if traceID == "" {
			traceID = GenerateTraceID()
		}
		w.Header().Set("X-Trace-ID", traceID)

		ctx := WithTraceID(r.Context(), traceID)
		Info(ctx, "request start",
			"method", r.Method,
			"path", r.URL.Path,
		)

		start := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))

		Info(ctx, "request end",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
