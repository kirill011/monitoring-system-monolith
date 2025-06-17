package services

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"fmt"
	"monolith/internal/models"
	"monolith/internal/repo"
)

type Auth interface {
	Create(ctx context.Context, params CreateUserParams) (CreateUserResult, error)
	Read(ctx context.Context) (ReadResult, error)
	Update(ctx context.Context, params UpdateUsersParams) error
	Delete(ctx context.Context, userID int32) error

	Authorize(ctx context.Context, params AuthorizeParams) (int, error)
	GetEmailsByIDs(ctx context.Context, userIDs []int32) ([]string, error)
}

type AuthService struct {
	repo repo.Auth
}

func NewAuthService(r repo.Auth) *AuthService {
	return &AuthService{
		repo: r,
	}
}

type (
	CreateUserParams struct {
		Name     string
		Email    string
		Password string
	}

	CreateUserResult struct {
		ID    int32
		Name  string
		Email string
	}

	UpdateUsersParams struct {
		ID       int32
		Name     *string
		Email    *string
		Password *string
	}

	ReadResult struct {
		Users []models.User
	}

	AuthorizeParams struct {
		Email    string
		Password string
	}
)

func (s *AuthService) Create(ctx context.Context, params CreateUserParams) (CreateUserResult, error) {
	h := sha1.New() //nolint:gosec
	h.Write([]byte(params.Password))
	passHash := hex.EncodeToString(h.Sum(nil))

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	ret, err := tx.Create(repo.CreateUserOpts{
		Name:     params.Name,
		Email:    params.Email,
		Password: passHash,
	})
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("tx.Create: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return CreateUserResult{
		ID:    ret.ID,
		Name:  ret.Name,
		Email: ret.Email,
	}, nil
}

func (s *AuthService) Read(ctx context.Context) (ReadResult, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return ReadResult{}, fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	ret, err := tx.Read(ctx)
	if err != nil {
		return ReadResult{}, fmt.Errorf("tx.Read: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return ReadResult{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return ReadResult{
		Users: ret.Users,
	}, nil
}

func (s *AuthService) Update(ctx context.Context, params UpdateUsersParams) error {
	if params.Password != nil {
		h := sha1.New() //nolint:gosec
		h.Write([]byte(*params.Password))
		passHash := hex.EncodeToString(h.Sum(nil))
		params.Password = &passHash
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	err = tx.Update(ctx, repo.UpdateUsersOpts{
		ID:       params.ID,
		Name:     params.Name,
		Email:    params.Email,
		Password: params.Password,
	})
	if err != nil {
		return fmt.Errorf("s.repo.Update: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}

func (s *AuthService) Delete(ctx context.Context, userID int32) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	err = tx.Delete(ctx, userID)
	if err != nil {
		return fmt.Errorf("tx.Delete: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}

func (s *AuthService) Authorize(ctx context.Context, params AuthorizeParams) (int, error) {
	h := sha1.New() //nolint:gosec
	h.Write([]byte(params.Password))
	passHash := hex.EncodeToString(h.Sum(nil))

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	userID, err := tx.Authorize(repo.AuthorizeOpts{
		Email:    params.Email,
		Password: passHash,
	})
	if err != nil {
		return 0, fmt.Errorf("tx.Authorize: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("tx.Commit: %w", err)
	}

	return userID, nil
}

func (s *AuthService) GetEmailsByIDs(ctx context.Context, userIDs []int32) ([]string, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("s.repo.BeginTx: %w", err)
	}
	defer tx.Rollback()

	ret, err := tx.GetEmailsByIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("tx.GetEmailByID: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("tx.Commit: %w", err)
	}

	return ret, nil
}
