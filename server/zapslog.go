package server

import (
	"log/slog"

	"go.uber.org/zap/zapcore"
)

type slogCore struct {
	h *slog.Logger
}

var _ zapcore.Core = &slogCore{}

func (c *slogCore) Enabled(_ zapcore.Level) bool { return true }
func (c *slogCore) Sync() error                  { return nil }

func (c *slogCore) With(fields []zapcore.Field) zapcore.Core {
	attrs := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		attrs = append(attrs, f.Key, f.Interface)
	}
	if c.h == nil {
		c.h = slog.Default()
	}
	return &slogCore{h: c.h.With(attrs...)}
}

func (c *slogCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *slogCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	attrs := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		attrs = append(attrs, f.Key, f.Interface)
	}

	if c.h == nil {
		c.h = slog.Default()
	}

	switch ent.Level {
	case zapcore.DebugLevel:
		c.h.Debug(ent.Message, attrs...)
	case zapcore.InfoLevel:
		c.h.Info(ent.Message, attrs...)
	case zapcore.WarnLevel:
		c.h.Warn(ent.Message, attrs...)
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		c.h.Error(ent.Message, attrs...)
	}

	return nil
}
