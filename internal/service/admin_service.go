package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"4vpx/internal/domain"
	"4vpx/internal/security"
	"4vpx/internal/storage/sqlite"
)

type AdminService struct {
	store *sqlite.Store
}

func NewAdminService(store *sqlite.Store) *AdminService {
	return &AdminService{store: store}
}

func (s *AdminService) Initialize(ctx context.Context, username, password string) (domain.Admin, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return domain.Admin{}, ErrEmptyUsername
	}
	if strings.TrimSpace(password) == "" {
		return domain.Admin{}, ErrEmptyPassword
	}

	count, err := s.store.Admins.Count(ctx)
	if err != nil {
		return domain.Admin{}, err
	}
	if count > 0 {
		return domain.Admin{}, ErrAdminAlreadyInitialized
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return domain.Admin{}, err
	}

	now := time.Now().UTC()
	return s.store.Admins.Create(ctx, domain.Admin{
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (s *AdminService) EnsureInitialized(ctx context.Context, username, password string) (domain.Admin, error) {
	count, err := s.store.Admins.Count(ctx)
	if err != nil {
		return domain.Admin{}, err
	}
	if count == 0 {
		return s.Initialize(ctx, username, password)
	}
	admins, err := s.store.Admins.List(ctx)
	if err != nil {
		return domain.Admin{}, err
	}
	if len(admins) == 0 {
		return domain.Admin{}, sql.ErrNoRows
	}
	return admins[0], nil
}

func (s *AdminService) Authenticate(ctx context.Context, username, password string) (domain.Admin, error) {
	admin, err := s.store.Admins.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Admin{}, ErrInvalidCredentials
		}
		return domain.Admin{}, err
	}
	if err := security.CheckPassword(admin.PasswordHash, password); err != nil {
		return domain.Admin{}, ErrInvalidCredentials
	}
	return admin, nil
}

func (s *AdminService) ChangePassword(ctx context.Context, adminID int64, currentPassword, newPassword string) error {
	if strings.TrimSpace(newPassword) == "" {
		return ErrEmptyPassword
	}

	admin, err := s.store.Admins.GetByID(ctx, adminID)
	if err != nil {
		return err
	}
	if err := security.CheckPassword(admin.PasswordHash, currentPassword); err != nil {
		return ErrInvalidCredentials
	}

	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.store.Admins.UpdatePasswordHash(ctx, adminID, passwordHash, time.Now().UTC())
}

func (s *AdminService) ResetCredentials(ctx context.Context, username, password string) (domain.Admin, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return domain.Admin{}, ErrEmptyUsername
	}
	if strings.TrimSpace(password) == "" {
		return domain.Admin{}, ErrEmptyPassword
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return domain.Admin{}, err
	}

	count, err := s.store.Admins.Count(ctx)
	if err != nil {
		return domain.Admin{}, err
	}
	if count == 0 {
		return s.Initialize(ctx, username, password)
	}

	admins, err := s.store.Admins.List(ctx)
	if err != nil {
		return domain.Admin{}, err
	}
	if len(admins) == 0 {
		return domain.Admin{}, sql.ErrNoRows
	}

	admin := admins[0]
	admin.Username = username
	admin.PasswordHash = passwordHash
	admin.UpdatedAt = time.Now().UTC()
	if err := s.store.Admins.UpdateCredentials(ctx, admin.ID, admin.Username, admin.PasswordHash, admin.UpdatedAt); err != nil {
		return domain.Admin{}, err
	}
	return admin, nil
}
