package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"monolith/internal/models"
	"monolith/internal/repo"
	"monolith/pkg/postgres"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgconn"
)

type tagsRepo struct {
	db *sqlx.DB
	tx *sqlx.Tx

	log *zap.Logger
}

func NewTagsRepo(p *postgres.Postgres, log *zap.Logger) repo.Tags {
	return &tagsRepo{
		db:  p.DB,
		log: log,
	}
}

func (r tagsRepo) BeginTx(ctx context.Context) (repo.Tags, error) {
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

func (r tagsRepo) Commit() error {
	err := r.tx.Commit()
	if err != nil {
		return fmt.Errorf("r.tx.Commit: %w", err)
	}

	return nil
}

func (r tagsRepo) Rollback() error {
	err := r.tx.Rollback()
	if err != nil {
		return fmt.Errorf("r.tx.Rollback: %w", err)
	}

	return nil
}

const tagsRepoQueryInsert = `
insert into tags (name, device_id, regexp, compare_type, value, array_index, subject, severity_level, created_at)
values
(:name, :device_id, :regexp, :compare_type, :value, :array_index, :subject, :severity_level, :created_at)
returning id, "name", device_id, regexp, compare_type, value, array_index, subject, severity_level, created_at, updated_at;
`

func (r tagsRepo) Create(opts models.Tag) (models.Tag, error) {
	query, args, err := sqlx.Named(tagsRepoQueryInsert,
		map[string]any{
			"name":           opts.Name,
			"device_id":      opts.DeviceId,
			"regexp":         opts.Regexp,
			"compare_type":   opts.CompareType,
			"value":          opts.Value,
			"array_index":    opts.ArrayIndex,
			"subject":        opts.Subject,
			"severity_level": opts.SeverityLevel,
			"created_at":     time.Now(),
		},
	)
	if err != nil {
		return models.Tag{}, fmt.Errorf("sqlx.Named: %w", err)
	}
	query = sqlx.Rebind(sqlx.BindType(r.tx.DriverName()), query)

	var tag models.Tag
	err = r.tx.Get(&tag, query, args...)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code == pgErrCodeUniqueViolation {
				return models.Tag{}, repo.ErrDeviceExists
			}
		}

		return models.Tag{}, fmt.Errorf("s.tx.Get: %w", err)
	}
	return tag, nil
}

const tagsRepoQueryRead = `
select id, "name", device_id, regexp, compare_type, value, array_index, subject, severity_level, created_at, updated_at from tags
where deleted_at is null;
`

func (r tagsRepo) Read(ctx context.Context) (repo.ReadTagsResult, error) {
	var result repo.ReadTagsResult
	err := r.tx.SelectContext(ctx, &result.Tags, tagsRepoQueryRead)
	if err != nil {
		return repo.ReadTagsResult{}, fmt.Errorf("r.tx.SelectContext: %w", err)
	}

	return result, nil
}

const tagsRepoQueryUpdate = `
update tags 
set name = coalesce(:name, name),
    device_id = coalesce(:device_id, device_id),
	regexp = coalesce(:regexp, regexp),
	compare_type = coalesce(:compare_type, compare_type),
	value = coalesce(:value, value),
	array_index = coalesce(:array_index, array_index),
	subject = coalesce(:subject, subject),
	severity_level = coalesce(:severity_level, severity_level),
	updated_at = :updated_at
where id = :id and deleted_at is null;
`

func (r tagsRepo) Update(ctx context.Context, opts repo.UpdateTagsOpts) error {
	_, err := r.tx.NamedExecContext(ctx, tagsRepoQueryUpdate,
		struct {
			ID            int32     `db:"id"`
			Name          *string   `db:"name"`
			DeviceId      *int32    `db:"device_id"`
			Regexp        *string   `db:"regexp"`
			CompareType   *string   `db:"compare_type"`
			Value         *string   `db:"value"`
			ArrayIndex    *int32    `db:"array_index"`
			Subject       *string   `db:"subject"`
			SeverityLevel *string   `db:"severity_level"`
			UpdatedAt     time.Time `db:"updated_at"`
		}{
			ID:            opts.ID,
			Name:          opts.Name,
			DeviceId:      opts.DeviceId,
			Regexp:        opts.Regexp,
			CompareType:   opts.CompareType,
			Value:         opts.Value,
			ArrayIndex:    opts.ArrayIndex,
			Subject:       opts.Subject,
			SeverityLevel: opts.SeverityLevel,
			UpdatedAt:     time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("s.tx.NamedExecContext: %w", err)
	}
	return nil
}

const tagsRepoQueryDelete = `
delete from tags
where id = :id and deleted_at is null;
`

func (r tagsRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.tx.NamedExecContext(ctx, tagsRepoQueryDelete,
		map[string]any{
			"id": id,
		},
	)
	if err != nil {
		return fmt.Errorf("s.tx.NamedExecContext: %w", err)
	}
	return nil
}
