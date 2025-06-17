package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

type Config struct {
	LogLevel    string
	ServiceName string
	LogPath     string
}

// New returned new *zap.Logger not sugared build instance
func New(cfg Config) *zap.Logger {
	// Get an opinionated EncoderConfig for production environments
	config := zap.NewProductionEncoderConfig()

	// Configuring EncoderConfig parameters
	config.MessageKey = "message"
	config.TimeKey = "timestamp"
	config.StacktraceKey = ""
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create a fast, low-allocation JSON encoders
	fileEncoder := zapcore.NewJSONEncoder(config)
	consoleEncoder := zapcore.NewJSONEncoder(config)

	// Determining the logging level
	level, err := zapcore.ParseLevel(cfg.LogLevel)
	if err != nil {
		return zap.New(zapcore.NewCore(consoleEncoder, os.Stdout, zap.DebugLevel))
	}

	// Create directory _logs if not exist
	if _, err = os.Stat(cfg.LogPath); os.IsNotExist(err) {
		if err = os.Mkdir(cfg.LogPath, 0o777); err != nil { //nolint:mnd
			return zap.New(zapcore.NewCore(consoleEncoder, os.Stdout, zap.DebugLevel))
		}
	}

	// Open log file by path _logs/{serviceName}.log
	filename := cfg.LogPath + "/" + cfg.ServiceName + ".log"
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:mnd
	if err != nil {
		return zap.New(zapcore.NewCore(consoleEncoder, os.Stdout, zap.DebugLevel))
	}

	// Create a Core that duplicates log entries into two underlying Cores
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(file), level),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
	)

	// Constructs a new Logger from the provided core
	return zap.New(core, zap.AddCaller())
}

// FromCtx returns the Logger associated with the ctx.
// If no logger is associated, a disabled logger is returned.
func FromCtx(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok { //nolint:varnamelen
		return logger
	}
	return zap.NewNop()
}

// WithCtx returns a copy of ctx with the Logger attached.
func WithCtx(ctx context.Context, logger *zap.Logger) context.Context {
	if existed, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok { //nolint:varnamelen
		if existed == logger {
			// Do not store same logger.
			return ctx
		}
	}
	return context.WithValue(ctx, ctxKey{}, logger)
}
