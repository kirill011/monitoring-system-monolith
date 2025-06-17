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

const pgErrCodeUniqueViolation = "23505"

type devicesRepo struct {
	db *sqlx.DB
	tx *sqlx.Tx

	log *zap.Logger
}

func NewDevicesRepo(p *postgres.Postgres, log *zap.Logger) repo.Devices {
	return &devicesRepo{
		db:  p.DB,
		log: log,
	}
}

func (r devicesRepo) BeginTx(ctx context.Context) (repo.Devices, error) {
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

func (r devicesRepo) Commit() error {
	err := r.tx.Commit()
	if err != nil {
		return fmt.Errorf("r.tx.Commit: %w", err)
	}

	return nil
}

func (r devicesRepo) Rollback() error {
	err := r.tx.Rollback()
	if err != nil {
		return fmt.Errorf("r.tx.Rollback: %w", err)
	}

	return nil
}

const devicesRepoQueryInsertPerson = `
insert into devices (device_type, "name", address, responsible, created_at)
values
(:device_type, :name, :address, :responsible, :created_at)
returning id, device_type, "name", address, responsible, created_at, updated_at;
`

func (r devicesRepo) Create(opts models.Device) (models.Device, error) {
	rows, err := r.tx.NamedQuery(devicesRepoQueryInsertPerson,
		map[string]any{
			"name":        opts.Name,
			"device_type": opts.DeviceType,
			"address":     opts.Address,
			"responsible": opts.Responsible,
			"created_at":  time.Now(),
		},
	)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code == pgErrCodeUniqueViolation {
				return models.Device{}, repo.ErrDeviceExists
			}
		}

		return models.Device{}, fmt.Errorf("s.tx.NamedQuery: %w", err)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to closing rows", zap.Error(err))
		}
	}()

	if !rows.Next() {
		return models.Device{}, repo.ErrDeviceNotFound
	}
	var device models.Device
	err = rows.StructScan(&device)
	if err != nil {
		return models.Device{}, fmt.Errorf("rows.Scan: %w", err)
	}

	return device, nil
}

const devicesRepoQueryRead = `
select id, device_type, "name", address, responsible, created_at, updated_at from devices
where deleted_at is null;
`

func (r devicesRepo) Read(ctx context.Context) (repo.ReadDevicesResult, error) {
	var result repo.ReadDevicesResult
	err := r.tx.SelectContext(ctx, &result.Devices, devicesRepoQueryRead)
	if err != nil {
		return repo.ReadDevicesResult{}, fmt.Errorf("r.tx.SelectContext: %w", err)
	}

	return result, nil
}

const devicesRepoQueryUpdate = `
update devices 
set name = coalesce(:name, name),
    device_type = coalesce(:device_type, device_type),
	address = coalesce(:address, address),
	responsible = coalesce(:responsible, responsible),
	updated_at = :updated_at
where id = :id and deleted_at is null;
`

func (r devicesRepo) Update(ctx context.Context, opts repo.UpdateDeviceOpts) error {
	var responsible []int32
	if len(opts.Responsible) != 0 {
		responsible = opts.Responsible
	}
	_, err := r.tx.NamedExecContext(ctx, devicesRepoQueryUpdate,
		struct {
			ID          int32     `db:"id"`
			Name        *string   `db:"name"`
			DeviceType  *string   `db:"device_type"`
			Address     *string   `db:"address"`
			Responsible []int32   `db:"responsible"`
			UpdatedAt   time.Time `db:"updated_at"`
		}{
			ID:          opts.ID,
			Name:        opts.Name,
			DeviceType:  opts.DeviceType,
			Address:     opts.Address,
			Responsible: responsible,
			UpdatedAt:   time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("s.tx.NamedExecContext: %w", err)
	}
	return nil
}

const devicesRepoQueryDelete = `
delete from devices
where id = :id and deleted_at is null;
`

func (r devicesRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.tx.NamedExecContext(ctx, devicesRepoQueryDelete,
		map[string]any{
			"id": id,
		},
	)
	if err != nil {
		return fmt.Errorf("s.tx.NamedExecContext: %w", err)
	}
	return nil
}

const devicesRepoQueryGetResponsible = `
select responsible from devices
where id = :id and deleted_at is null;
`

func (r devicesRepo) GetResponsible(ctx context.Context, deviceID int32) ([]int32, error) {
	var responsibleIds models.SqlJsonbIntArray

	rows, err := r.tx.NamedQuery(devicesRepoQueryGetResponsible,
		map[string]any{
			"id": deviceID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("r.tx.NamedQuery: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, repo.ErrDeviceNotFound
	}

	err = rows.Scan(&responsibleIds)
	if err != nil {
		return nil, fmt.Errorf("rows.Scan: %w", err)
	}

	return responsibleIds, nil
}
