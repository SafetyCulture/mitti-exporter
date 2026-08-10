package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/SafetyCulture/mitti-exporter/internal/app/version"
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/diagnostics"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

var (
	infoStr      = "%s\n[info] "
	warnStr      = "%s\n[warn] "
	errStr       = "%s\n[error] "
	traceStr     = "%s\n[%.3fms] [rows:%v] %s"
	traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
	traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	trace        = false
)

// GormLogger wraps logger with some gorm specific functionality
type GormLogger struct {
	*zap.SugaredLogger

	SlowThreshold time.Duration
}

// LogMode log mode.
func (l *GormLogger) LogMode(logger.LogLevel) logger.Interface {
	newlogger := *l

	return &newlogger
}

// Info print info.
func (l GormLogger) Info(_ context.Context, msg string, data ...interface{}) {
	l.Infof(infoStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
}

// Warn print warn messages.
func (l GormLogger) Warn(_ context.Context, msg string, data ...interface{}) {
	l.Warnf(warnStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
}

// Error print error messages.
func (l GormLogger) Error(_ context.Context, msg string, data ...interface{}) {
	l.Errorf(errStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
}

// Trace print sql message.
func (l GormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	// do not log ErrRecordNotFound errors
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil:
		sql, rows := fc()
		if rows == -1 {
			l.Errorf(traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Errorf(traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.Debugf(traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Debugf(traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	default:
		if !trace {
			return
		}

		sql, rows := fc()

		if rows == -1 {
			l.Debugf(traceStr, utils.FileWithLineNum(), "-", sql)
		} else {
			l.Debugf(traceStr, utils.FileWithLineNum(), rows, sql)
		}
	}
}

var slg *zap.SugaredLogger

func getLogger(cores ...zapcore.Core) *zap.SugaredLogger {
	// Log to both console and the log file. This allows for succinct console logs and
	// Verbose detailed logs to review is something goes wrong.
	core := zapcore.NewTee(
		cores...,
	)

	// From a zapcore.Core, it's easy to construct a Logger.
	l := zap.New(core).Named(version.GetIntegrationID())
	defer l.Sync()

	// redirects output from the standard library's package-global logger to the supplied logger
	zap.RedirectStdLog(l)

	slg := l.Sugar().
		With(
			"version", version.GetVersion(),
			"pid", os.Getpid(),
			"uid", os.Getuid(),
		)

	if sysInfo, err := diagnostics.GetSysInfo(); err != nil {
		slg.With("err", err).Error("failed to get system info")
	} else {
		slg.With("sys_info", sysInfo).
			Debug("system info")
	}

	return slg
}

// GetLogger returns a configured instance of the logger
func GetLogger() *zap.SugaredLogger {
	if slg != nil {
		return slg
	}

	prodConfig := zap.NewProductionEncoderConfig()
	prodConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logFileEncoder := zapcore.NewJSONEncoder(prodConfig)
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	logName := fmt.Sprintf("sc-exporter-%s.log", time.Now().Format(time.DateOnly))
	file, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("unable to open log file %v", err)
	}

	logFileWriter := zapcore.Lock(file)
	consoleWriter := zapcore.Lock(os.Stderr)

	slg = getLogger(
		zapcore.NewCore(consoleEncoder, consoleWriter, zap.InfoLevel),
		zapcore.NewCore(logFileEncoder, logFileWriter, zap.DebugLevel),
	)

	return slg
}

// GetExporterLogger returns a configured instance of the logger (used by the UI)
func GetExporterLogger(path string) *ExporterLogger {
	if slg != nil {
		return &ExporterLogger{
			l: slg,
		}
	}

	prodConfig := zap.NewProductionEncoderConfig()
	prodConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logFileEncoder := zapcore.NewJSONEncoder(prodConfig)

	logName := fmt.Sprintf("sc-exporter-%s.log", time.Now().Format(time.DateOnly))
	file, err := os.OpenFile(filepath.Join(path, logName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("unable to open log file %v", err)
	}

	logFileWriter := zapcore.Lock(file)

	slg = getLogger(zapcore.NewCore(logFileEncoder, logFileWriter, zap.DebugLevel))

	return &ExporterLogger{
		l: slg,
	}
}

// ExporterLogger wraps sugared logged into an interface compatible for Wails
type ExporterLogger struct {
	l *zap.SugaredLogger
}

func (logger *ExporterLogger) Debug(message string) {
	logger.l.Debugln(message)
}

func (logger *ExporterLogger) Info(message string) {
	logger.l.Infoln(message)
}

func (logger *ExporterLogger) Warning(message string) {
	logger.l.Warnln(message)
}

func (logger *ExporterLogger) Error(message string) {
	logger.l.Errorln(message)
}

func (logger *ExporterLogger) Fatal(message string) {
	logger.l.Fatalln(message)
}

func (logger *ExporterLogger) Print(string) {
	panic("don't use print")
}

func (logger *ExporterLogger) Trace(string) {
	// unimplemented
	panic("don't use trace")
}
