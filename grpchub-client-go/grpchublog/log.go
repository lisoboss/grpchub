package grpchublog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"google.golang.org/grpc/grpclog"
)

type slogLogger struct {
	logger *slog.Logger
	tag    string
}

// --------- LoggerV2 Implementation ---------
func (l *slogLogger) log(level slog.Level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Log(context.Background(), level, msg, "component", l.tag, "caller", caller(3))
}

func (l *slogLogger) Info(args ...any)                 { l.log(slog.LevelInfo, "%s", fmt.Sprint(args...)) }
func (l *slogLogger) Infoln(args ...any)               { l.Info(args...) }
func (l *slogLogger) Infof(format string, args ...any) { l.log(slog.LevelInfo, format, args...) }

func (l *slogLogger) Warning(args ...any)                 { l.log(slog.LevelWarn, "%s", fmt.Sprint(args...)) }
func (l *slogLogger) Warningln(args ...any)               { l.Warning(args...) }
func (l *slogLogger) Warningf(format string, args ...any) { l.log(slog.LevelWarn, format, args...) }

func (l *slogLogger) Error(args ...any)                 { l.log(slog.LevelError, "%s", fmt.Sprint(args...)) }
func (l *slogLogger) Errorln(args ...any)               { l.Error(args...) }
func (l *slogLogger) Errorf(format string, args ...any) { l.log(slog.LevelError, format, args...) }

func (l *slogLogger) Fatal(args ...any) {
	l.log(slog.LevelError, "%s", fmt.Sprint(args...))
	os.Exit(1)
}
func (l *slogLogger) Fatalln(args ...any) { l.Fatal(args...) }
func (l *slogLogger) Fatalf(format string, args ...any) {
	l.log(slog.LevelError, format, args...)
	os.Exit(1)
}

func (l *slogLogger) V(_ int) bool {
	// Always verbose for development; adjust as needed
	return true
}

// --------- Component Logger Factory ---------

// Component returns a grpclog.LoggerV2 for a specific component.
func Component(name string) grpclog.LoggerV2 {
	return &slogLogger{
		logger: slog.Default(),
		tag:    name,
	}
}

// InitGRPCLogger sets the slog logger as gRPC default logger.
func InitGRPCLogger() {
	grpclog.SetLoggerV2(Component("grpc"))
}

// --------- Helper ---------

// caller returns file:line of the caller.
func caller(depth int) string {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", file, line)
}
