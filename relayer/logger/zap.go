package logger

import "go.uber.org/zap"

// Logger is a logger interface.
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

// MetricRecorder is coreum metric recorder interface.
type MetricRecorder interface {
	IncrementError()
}

// ZapLogger is logger wrapper with an ability to add error logs metric record.
type ZapLogger struct {
	*zap.Logger
	metricRecorder MetricRecorder
}

// NewZapLogger returns a new instance of the ZapLogger.
func NewZapLogger(log *zap.Logger, metricRecorder MetricRecorder) *ZapLogger {
	return &ZapLogger{
		Logger:         log,
		metricRecorder: metricRecorder,
	}
}

// Error logs a message at ErrorLevel and add error metric record.
func (log *ZapLogger) Error(msg string, fields ...zap.Field) {
	log.Logger.Error(msg, fields...)
	log.metricRecorder.IncrementError()
}
