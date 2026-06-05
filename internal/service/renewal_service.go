package service

import (
	"context"
	"time"

	"4vpx/internal/domain"
	"4vpx/internal/storage/sqlite"
)

type RenewalService struct {
	store *sqlite.Store
}

func NewRenewalService(store *sqlite.Store) *RenewalService {
	return &RenewalService{store: store}
}

func (s *RenewalService) RenewDays(ctx context.Context, userID int64, days int, actor, notes string, now time.Time) (domain.User, domain.RenewalRecord, error) {
	if days <= 0 {
		return domain.User{}, domain.RenewalRecord{}, ErrInvalidRenewalDays
	}
	return s.applyRenewal(ctx, userID, actor, notes, now, "renew_days", func(base time.Time) time.Time {
		return base.AddDate(0, 0, days)
	})
}

func (s *RenewalService) RenewMonth(ctx context.Context, userID int64, actor, notes string, now time.Time) (domain.User, domain.RenewalRecord, error) {
	return s.applyRenewal(ctx, userID, actor, notes, now, "renew_month", func(base time.Time) time.Time {
		return addMonthsClamped(base, 1)
	})
}

func (s *RenewalService) ExtendTo(ctx context.Context, userID int64, target time.Time, actor, notes string, now time.Time) (domain.User, domain.RenewalRecord, error) {
	if target.IsZero() {
		return domain.User{}, domain.RenewalRecord{}, ErrInvalidExpiry
	}

	var updatedUser domain.User
	var record domain.RenewalRecord
	err := s.store.WithTx(ctx, func(tx *sqlite.Store) error {
		user, err := tx.Users.GetByID(ctx, userID)
		if err != nil {
			return err
		}

		nowUTC := now.UTC()
		before := user.ExpiresAt.UTC()
		base := effectiveRenewalBase(before, nowUTC)
		targetUTC := target.UTC()
		if !targetUTC.After(base) {
			return ErrInvalidRenewalTarget
		}

		user.ExpiresAt = targetUTC
		user.UpdatedAt = nowUTC
		if err := tx.Users.Update(ctx, user); err != nil {
			return err
		}

		createdRecord, err := tx.Renewals.Create(ctx, domain.RenewalRecord{
			UserID:          user.ID,
			Days:            wholeDaysBetween(base, targetUTC),
			Action:          "renew_until",
			BeforeExpiresAt: before,
			AfterExpiresAt:  targetUTC,
			Notes:           notes,
			Actor:           actor,
			CreatedAt:       nowUTC,
		})
		if err != nil {
			return err
		}

		updatedUser = user
		updatedUser.RemainingDays = remainingDays(user.ExpiresAt, nowUTC)
		record = createdRecord
		return nil
	})
	if err != nil {
		return domain.User{}, domain.RenewalRecord{}, err
	}
	return updatedUser, record, nil
}

func (s *RenewalService) ListByUserID(ctx context.Context, userID int64) ([]domain.RenewalRecord, error) {
	return s.store.Renewals.ListByUserID(ctx, userID)
}

func (s *RenewalService) applyRenewal(ctx context.Context, userID int64, actor, notes string, now time.Time, action string, nextExpiry func(base time.Time) time.Time) (domain.User, domain.RenewalRecord, error) {
	var updatedUser domain.User
	var record domain.RenewalRecord
	err := s.store.WithTx(ctx, func(tx *sqlite.Store) error {
		user, err := tx.Users.GetByID(ctx, userID)
		if err != nil {
			return err
		}

		nowUTC := now.UTC()
		before := user.ExpiresAt.UTC()
		base := effectiveRenewalBase(before, nowUTC)
		after := nextExpiry(base).UTC()

		user.ExpiresAt = after
		user.UpdatedAt = nowUTC
		if err := tx.Users.Update(ctx, user); err != nil {
			return err
		}

		createdRecord, err := tx.Renewals.Create(ctx, domain.RenewalRecord{
			UserID:          user.ID,
			Days:            wholeDaysBetween(base, after),
			Action:          action,
			BeforeExpiresAt: before,
			AfterExpiresAt:  after,
			Notes:           notes,
			Actor:           actor,
			CreatedAt:       nowUTC,
		})
		if err != nil {
			return err
		}

		updatedUser = user
		updatedUser.RemainingDays = remainingDays(user.ExpiresAt, nowUTC)
		record = createdRecord
		return nil
	})
	if err != nil {
		return domain.User{}, domain.RenewalRecord{}, err
	}
	return updatedUser, record, nil
}
