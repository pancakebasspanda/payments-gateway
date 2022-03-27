package postgres

import (
	"context"

	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
)

type pgLogger struct{}

func (p *pgLogger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	logger := log.WithFields(data).WithField("PgxStorage", "pgx")

	switch level {
	case pgx.LogLevelTrace:
		logger.WithField("pgx_log_level", level).Debug(msg)
	case pgx.LogLevelDebug:
		logger.Debug(msg)
	case pgx.LogLevelInfo:
		logger.Info(msg)
	case pgx.LogLevelWarn:
		logger.Warn(msg)
	case pgx.LogLevelError:
		logger.Error(msg)
	default:
		logger.WithField("invalid_pgx_log_level", level).Error(msg)
	}
}
