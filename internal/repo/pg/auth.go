package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"monolith/internal/repo"
	"monolith/pkg/postgres"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgconn"
)

type authRepo struct {
	db *sqlx.DB
	tx *sqlx.Tx

	log *zap.Logger
}

func NewAuthRepo(p *postgres.Postgres, log *zap.Logger) repo.Auth {
	return &authRepo{
		db:  p.DB,
		log: log,
	}
}

func (r authRepo) BeginTx(ctx context.Context) (repo.Auth, error) {
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

func (r authRepo) Commit() error {
	err := r.tx.Commit()
	if err != nil {
		return fmt.Errorf("r.tx.Commit: %w", err)
	}

	return nil
}

func (r authRepo) Rollback() error {
	err := r.tx.Rollback()
	if err != nil {
		return fmt.Errorf("r.tx.Rollback: %w", err)
	}

	return nil
}

const usersRepoQueryInsertPerson = `
insert into users (name, email, password, created_at)
values
(:name, :email, :password, :created_at)
returning id, name, email;
`

func (r authRepo) Create(opts repo.CreateUserOpts) (repo.CreateUserResult, error) {
	rows, err := r.tx.NamedQuery(usersRepoQueryInsertPerson,
		struct {
			Name      string    `db:"name"`
			Email     string    `db:"email"`
			Password  string    `db:"password"`
			CreatedAt time.Time `db:"created_at"`
		}{
			Name:      opts.Name,
			Email:     opts.Email,
			Password:  opts.Password,
			CreatedAt: time.Now(),
		},
	)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code == pgErrCodeUniqueViolation {
				return repo.CreateUserResult{}, repo.ErrUserExists
			}
		}

		return repo.CreateUserResult{}, fmt.Errorf("s.tx.NamedQuery: %w", err)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to closing rows", zap.Error(err))
		}
	}()

	if !rows.Next() {
		return repo.CreateUserResult{}, repo.ErrUsersNotFound
	}
	var user repo.CreateUserResult
	err = rows.StructScan(&user)
	if err != nil {
		return repo.CreateUserResult{}, fmt.Errorf("rows.Scan: %w", err)
	}

	return user, nil
}

const usersRepoQuerySelectUsers = `
select id, name, email, created_at, updated_at from users
where deleted_at is null;
`

func (r authRepo) Read(ctx context.Context) (repo.ReadUsersResult, error) {
	var result repo.ReadUsersResult
	err := r.tx.SelectContext(ctx, &result.Users, usersRepoQuerySelectUsers)
	if err != nil {
		return repo.ReadUsersResult{}, fmt.Errorf("r.tx.SelectContext: %w", err)
	}

	return result, nil
}

const usersRepoQueryUpdateUser = `
update users 
set name = coalesce(:name, name),
	email = coalesce(:email, email),
	password = coalesce(:password, password),
	updated_at = :updated_at
where id = :id and deleted_at is null;
`

func (r authRepo) Update(ctx context.Context, opts repo.UpdateUsersOpts) error {
	_, err := r.tx.NamedExecContext(ctx, usersRepoQueryUpdateUser,
		struct {
			ID        int32     `db:"id"`
			Name      *string   `db:"name"`
			Email     *string   `db:"email"`
			Password  *string   `db:"password"`
			UpdatedAt time.Time `db:"updated_at"`
		}{
			ID:        opts.ID,
			Name:      opts.Name,
			Email:     opts.Email,
			Password:  opts.Password,
			UpdatedAt: time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("s.tx.NamedExecContext: %w", err)
	}
	return nil
}

const usersRepoQueryDeletePerson = `
delete from users
where id = :id and deleted_at is null;
`

func (r authRepo) Delete(ctx context.Context, userID int32) error {
	_, err := r.tx.NamedExecContext(ctx, usersRepoQueryDeletePerson,
		struct {
			ID int32 `db:"id"`
		}{
			ID: userID,
		},
	)
	if err != nil {
		return fmt.Errorf("s.tx.NamedExecContext: %w", err)
	}
	return nil
}

const usersRepoQueryAuthorizePerson = `
select id from users
where email = :email and password = :password and deleted_at is null
limit 1;
`

func (r authRepo) Authorize(opts repo.AuthorizeOpts) (int, error) {
	rows, err := r.tx.NamedQuery(usersRepoQueryAuthorizePerson,
		struct {
			Email    string `db:"email"`
			Password string `db:"password"`
		}{
			Email:    opts.Email,
			Password: opts.Password,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("s.tx.NamedQuery: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to closing rows", zap.Error(err))
		}
	}()

	if !rows.Next() {
		return 0, repo.ErrUsersNotFound
	}

	var userID int

	err = rows.Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("rows.Scan: %w", err)
	}

	return userID, nil
}

const usersRepoQueryGetEmailByID = `
select email from users
where id in (:users_ids) and deleted_at is null`

func (r authRepo) GetEmailsByIDs(ctx context.Context, userIDs []int32) ([]string, error) {
	var emails []string

	query, args, err := sqlx.Named(usersRepoQueryGetEmailByID,
		map[string]any{
			"users_ids": userIDs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("sqlx.Named: %w", err)
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlx.In: %w", err)
	}
	query = r.tx.Rebind(query)

	err = r.tx.SelectContext(ctx, &emails, query, args...)
	if err != nil {
		return nil, fmt.Errorf("r.tx.GetContext: %w", err)
	}

	return emails, nil
}
