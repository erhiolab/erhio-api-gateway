package logger

import (
	"context"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/utils"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Log 系统日志记录器
	Log *zap.Logger
	// RequestLog 请求日志记录器
	RequestLog *zap.Logger
)

// InitLogger 初始化日志
func InitLogger() {
	cfg := config.Get().Logger

	// 系统日志
	Log = createLogger(cfg.LogPath, cfg)
	// 请求日志
	RequestLog = createLogger(cfg.RequestLogPath, cfg)
}

// createLogger 创建日志记录器
func createLogger(logPath string, cfg config.LoggerConfig) *zap.Logger {
	// 配置文件输出
	writeSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})
	// 配置控制台输出
	consoleSyncer := zapcore.AddSync(os.Stdout)
	// 配置编码器
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.LevelKey = "level"
	encoderCfg.MessageKey = "msg"
	// 创建核心, 同时输出到文件和控制台
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), writeSyncer, zapcore.InfoLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), consoleSyncer, zapcore.DebugLevel),
	)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

// WithRequestLogCtx 从 context 中获取 requestID 并添加到日志中
func WithRequestLogCtx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return RequestLog
	}
	// 从 context 中获取 requestID
	if id, ok := ctx.Value(utils.RequestIDKey).(string); ok {
		return RequestLog.With(zap.String("request_id", id))
	}
	return RequestLog
}
