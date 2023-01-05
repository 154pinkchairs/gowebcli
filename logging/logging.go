package logging

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func SetupLogging() *zap.SugaredLogger {
	f, err := os.OpenFile("gowebcli.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	loglevel := os.Getenv("LOG_LEVEL")
	var zapLevel zapcore.Level
	switch loglevel {
	case "trace":
		zapLevel = zap.DebugLevel
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	case "fatal":
		zapLevel = zap.FatalLevel
	case "panic":
		zapLevel = zap.PanicLevel
	default:
		zapLevel = zap.WarnLevel
	}
	jsonEncoder := zap.NewDevelopmentEncoderConfig()
	//use a human readable time format (01 Jan 06 15:04:13.12345 MST)
	jsonEncoder.EncodeTime = zapcore.TimeEncoderOfLayout("02 Jan 06 15:04:13.12345 MST")
	jsonEncoder.EncodeLevel = zapcore.CapitalLevelEncoder
	jsonEncoder.EncodeCaller = zapcore.ShortCallerEncoder
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(jsonEncoder),
		zapcore.AddSync(f),
		zap.NewAtomicLevelAt(zapLevel),
	),
		zap.AddCaller(),
		//zap.AddCallerSkip(1),
	)
	Log := logger.Sugar()
	return Log
}
