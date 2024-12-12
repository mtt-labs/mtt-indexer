package logger

import (
	"fmt"
	"log"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

func init() {
	hook := &lumberjack.Logger{
		Filename:   fmt.Sprintf("log.log"),
		MaxSize:    50,
		MaxBackups: 5,
		MaxAge:     1,
		Compress:   false,
	}
	defer hook.Close()
	enConfig := zap.NewProductionEncoderConfig()

	enConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	level := zap.DebugLevel
	w := zapcore.AddSync(hook)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(enConfig),
		w,
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	_log := log.New(hook, "", log.LstdFlags)
	Logger = logger.Sugar()
	_log.Println("Start...")
}
