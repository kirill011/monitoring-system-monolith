package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"monolith/internal/models"
	"monolith/internal/repo"
	"monolith/pkg/postgres"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type messagesRepo struct {
	db *sqlx.DB
	tx *sqlx.Tx

	log *zap.Logger
}

func NewMessagesRepo(p *postgres.Postgres, log *zap.Logger) repo.Messages {
	return &messagesRepo{
		db:  p.DB,
		log: log,
	}
}

func (r messagesRepo) BeginTx(ctx context.Context) (repo.Messages, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: 0,
		ReadOnly:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("r.db.BeginTx: %w", err)
	}

	r.tx = tx

	return r, nil
}

func (r messagesRepo) Commit() error {
	err := r.tx.Commit()
	if err != nil {
		return fmt.Errorf("r.tx.Commit: %w", err)
	}

	return nil
}

func (r messagesRepo) Rollback() error {
	err := r.tx.Rollback()
	if err != nil {
		return fmt.Errorf("r.tx.Rollback: %w", err)
	}

	return nil
}

const messagesRepoQueryInsert = `
insert into messages (got_at, device_id, message, message_type, severity_level, component)
values
(:got_at, :device_id, :message, :message_type, :severity_level, :component)
`

func (r messagesRepo) Create(opts models.Message) error {
	_, err := r.tx.NamedExec(messagesRepoQueryInsert,
		map[string]any{
			"got_at":         time.Now(),
			"device_id":      opts.DeviceId,
			"message":        opts.Message,
			"message_type":   opts.MessageType,
			"severity_level": opts.SeverityLevel,
			"component":      opts.Component,
		},
	)
	if err != nil {
		return fmt.Errorf("s.tx.NamedExec: %w", err)
	}

	return nil
}

const messagesRepoQueryGetAllByPeriod = `
select got_at, device_id, message, message_type 
from messages
Where got_at between :start and :end
order by got_at desc
`

func (r messagesRepo) GetAllByPeriod(opts repo.MessagesGetAllByPeriodOpts) ([]models.Message, error) {
	messages := make([]models.Message, 0)

	query, args, err := sqlx.Named(messagesRepoQueryGetAllByPeriod, map[string]any{
		"start": opts.StartTime,
		"end":   opts.EndTime,
	})
	if err != nil {
		return nil, fmt.Errorf("sqlx.Named: %w", err)
	}

	query = sqlx.Rebind(sqlx.BindType(r.tx.DriverName()), query)

	err = r.tx.Select(&messages, query, args...)
	if err != nil {
		return nil, fmt.Errorf("r.tx.Select: %w", err)
	}

	return messages, nil
}

const messagesRepoQueryGetAllByDeviceId = `
select got_at, device_id, message, message_type
from messages
where device_id = :device_id
order by got_at desc
`

func (r messagesRepo) GetAllByDeviceId(deviceID int32) ([]models.Message, error) {
	messages := make([]models.Message, 0)

	query, args, err := sqlx.Named(messagesRepoQueryGetAllByDeviceId, map[string]any{
		"device_id": deviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("sqlx.Named: %w", err)
	}
	query = sqlx.Rebind(sqlx.BindType(r.tx.DriverName()), query)
	err = r.tx.Select(&messages, query, args...)
	if err != nil {
		return nil, fmt.Errorf("r.tx.Select: %w", err)
	}

	return messages, nil
}

const messagesRepoQueryGetCountByMessageType = `
select device_id, count(*) as count
from messages
where message_type = :message_type
group by device_id
`

func (r messagesRepo) GetCountByMessageType(messageType string) (repo.GetCountByMessageTypeResult, error) {
	counts := make([]models.CountByDeviceID, 0)

	query, args, err := sqlx.Named(messagesRepoQueryGetCountByMessageType,
		struct {
			MessageType string `db:"message_type"`
		}{
			MessageType: messageType,
		})
	if err != nil {
		return repo.GetCountByMessageTypeResult{}, fmt.Errorf("sqlx.Named: %w", err)
	}
	query = sqlx.Rebind(sqlx.BindType(r.tx.DriverName()), query)
	err = r.tx.Select(&counts, query, args...)
	if err != nil {
		return repo.GetCountByMessageTypeResult{}, fmt.Errorf("r.tx.Select: %w", err)
	}

	return repo.GetCountByMessageTypeResult{
		Count: counts,
	}, nil
}

const messagesRepoQueryMonthReport = `
SELECT 
    device_id,
    message_type,
    active_days,
    total_messages,
    avg_daily_messages,
    max_daily_messages,
    median_daily_messages,
    total_critical,
    max_daily_critical,
    max_daily_components,
    most_active_component,
    first_critical_time,
    last_critical_time,
    avg_critical_interval_sec,
    critical_percentage,
    overall_volume_rank
FROM (
    SELECT 
        *,
        DENSE_RANK() OVER (ORDER BY total_messages DESC) AS overall_volume_rank
    FROM (
        SELECT
            m.device_id,
            m.message_type,
            m.active_days,
            m.total_messages,
            m.avg_daily_messages,
            m.max_daily_messages,
            m.median_daily_messages,
            m.total_critical,
            m.max_daily_critical,
            m.max_daily_components,
            m.most_active_component,
            e.first_critical_time,
            e.last_critical_time,
            e.avg_critical_interval_sec,
            ROUND(100.0 * m.total_critical / NULLIF(m.total_messages, 0), 2) AS critical_percentage
        FROM (
            SELECT
                d.device_id,
                d.message_type,
                COUNT(DISTINCT d.day) AS active_days,
                SUM(d.total_messages) AS total_messages,
                AVG(d.total_messages) AS avg_daily_messages,
                MAX(d.total_messages) AS max_daily_messages,
                PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY d.total_messages) AS median_daily_messages,
                SUM(d.critical_count) AS total_critical,
                MAX(d.critical_count) AS max_daily_critical,
                MAX(d.unique_components) AS max_daily_components,
                (SELECT component FROM (
                    SELECT component, COUNT(*) 
                    FROM messages 
                    WHERE device_id = d.device_id 
                      AND message_type = d.message_type 
                      AND component IS NOT NULL
                    GROUP BY component
                    ORDER BY COUNT(*) DESC 
                    LIMIT 1
                ) sub_comp) AS most_active_component
            FROM (
                SELECT 
                    device_id,
                    DATE_TRUNC('day', got_at) AS day,
                    message_type,
                    COUNT(*) AS total_messages,
                    SUM(CASE severity_level WHEN 'critical' THEN 1 ELSE 0 END) AS critical_count,
                    COUNT(DISTINCT component) AS unique_components
                FROM messages
                WHERE got_at BETWEEN (NOW() - '30 days'::INTERVAL) AND NOW()
                GROUP BY device_id, DATE_TRUNC('day', got_at), message_type
            ) d
            GROUP BY d.device_id, d.message_type
        ) m
        LEFT JOIN (
            SELECT
                device_id,
                message_type,
                MIN(got_at) AS first_critical_time,
                MAX(got_at) AS last_critical_time,
                AVG(time_diff) AS avg_critical_interval_sec
            FROM (
                SELECT 
                    device_id,
                    message_type,
                    got_at,
                    EXTRACT(EPOCH FROM (got_at - LAG(got_at) OVER (
                        PARTITION BY device_id, message_type 
                        ORDER BY got_at
                    ))) AS time_diff
                FROM messages
                WHERE severity_level = 'critical'
                  AND got_at > (CURRENT_DATE - 30)
            ) critical_intervals
            GROUP BY device_id, message_type
        ) e ON m.device_id = e.device_id AND m.message_type = e.message_type
        WHERE m.total_messages > 100
    ) combined_data
) final_data
ORDER BY device_id, total_messages DESC;
`

func (r messagesRepo) MonthReport() ([]models.MonthReportRow, error) {
	result := make([]models.MonthReportRow, 0)

	err := r.tx.Select(&result, messagesRepoQueryMonthReport)
	if err != nil {
		return nil, fmt.Errorf("r.tx.Select: %w", err)
	}

	return result, nil
}
