package service

import (
	"context"
	"strings"
	"time"

	"4vpx/internal/domain"
	"4vpx/internal/security"
	"4vpx/internal/storage/sqlite"
)

type CreateUserInput struct {
	Name        string
	Notes       string
	Enabled     bool
	ExpiresAt   time.Time
	DeviceSlots int
}

type UpdateUserInput struct {
	Name      string
	Notes     string
	Enabled   bool
	ExpiresAt time.Time
}

type UserService struct {
	store *sqlite.Store
}

func NewUserService(store *sqlite.Store) *UserService {
	return &UserService{store: store}
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (domain.User, error) {
	if err := validateCreateUserInput(input); err != nil {
		return domain.User{}, err
	}

	accessToken, err := security.NewAccessToken()
	if err != nil {
		return domain.User{}, err
	}

	now := time.Now().UTC()
	user := domain.User{
		Name:        strings.TrimSpace(input.Name),
		Notes:       input.Notes,
		Enabled:     input.Enabled,
		ExpiresAt:   input.ExpiresAt.UTC(),
		AccessToken: accessToken,
		DeviceSlots: input.DeviceSlots,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = s.store.WithTx(ctx, func(tx *sqlite.Store) error {
		createdUser, err := tx.Users.Create(ctx, user)
		if err != nil {
			return err
		}
		user = createdUser

		for slotIndex := 1; slotIndex <= input.DeviceSlots; slotIndex++ {
			slot, err := newDeviceSlot(user.ID, slotIndex, now)
			if err != nil {
				return err
			}
			if _, err := tx.DeviceSlots.Create(ctx, slot); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.User{}, err
	}

	return s.populateRemainingDays(user, now), nil
}

func (s *UserService) Update(ctx context.Context, userID int64, input UpdateUserInput) (domain.User, error) {
	if err := validateUpdateUserInput(input); err != nil {
		return domain.User{}, err
	}

	user, err := s.store.Users.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	user.Name = strings.TrimSpace(input.Name)
	user.Notes = input.Notes
	user.Enabled = input.Enabled
	user.ExpiresAt = input.ExpiresAt.UTC()
	user.UpdatedAt = time.Now().UTC()

	if err := s.store.Users.Update(ctx, user); err != nil {
		return domain.User{}, err
	}
	return s.populateRemainingDays(user, time.Now().UTC()), nil
}

func (s *UserService) SetEnabled(ctx context.Context, userID int64, enabled bool) (domain.User, error) {
	user, err := s.store.Users.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	user.Enabled = enabled
	user.UpdatedAt = time.Now().UTC()

	if err := s.store.Users.Update(ctx, user); err != nil {
		return domain.User{}, err
	}
	return s.populateRemainingDays(user, time.Now().UTC()), nil
}

func (s *UserService) SetExpiresAt(ctx context.Context, userID int64, expiresAt time.Time) (domain.User, error) {
	if expiresAt.IsZero() {
		return domain.User{}, ErrInvalidExpiry
	}

	user, err := s.store.Users.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	user.ExpiresAt = expiresAt.UTC()
	user.UpdatedAt = time.Now().UTC()

	if err := s.store.Users.Update(ctx, user); err != nil {
		return domain.User{}, err
	}
	return s.populateRemainingDays(user, time.Now().UTC()), nil
}

func (s *UserService) Get(ctx context.Context, userID int64, now time.Time) (domain.User, error) {
	user, err := s.store.Users.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	return s.populateRemainingDays(user, now), nil
}

func (s *UserService) GetByAccessToken(ctx context.Context, token string, now time.Time) (domain.User, error) {
	user, err := s.store.Users.GetByAccessToken(ctx, token)
	if err != nil {
		return domain.User{}, err
	}
	return s.populateRemainingDays(user, now), nil
}

func (s *UserService) List(ctx context.Context, now time.Time) ([]domain.User, error) {
	users, err := s.store.Users.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		users[i] = s.populateRemainingDays(users[i], now)
	}
	return users, nil
}

func (s *UserService) Delete(ctx context.Context, userID int64) error {
	return s.store.Users.Delete(ctx, userID)
}

func (s *UserService) populateRemainingDays(user domain.User, now time.Time) domain.User {
	user.RemainingDays = remainingDays(user.ExpiresAt, now)
	return user
}

func validateCreateUserInput(input CreateUserInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return ErrEmptyUserName
	}
	if input.ExpiresAt.IsZero() {
		return ErrInvalidExpiry
	}
	if input.DeviceSlots <= 0 {
		return ErrInvalidDeviceSlotCount
	}
	return nil
}

func validateUpdateUserInput(input UpdateUserInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return ErrEmptyUserName
	}
	if input.ExpiresAt.IsZero() {
		return ErrInvalidExpiry
	}
	return nil
}
