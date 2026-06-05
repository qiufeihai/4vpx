package sqlite

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Store struct {
	db          *sql.DB
	Admins      *AdminRepository
	Sessions    *SessionRepository
	Users       *UserRepository
	DeviceSlots *DeviceSlotRepository
	Renewals    *RenewalRepository
	System      *SystemConfigRepository
}

func NewStore(db *sql.DB) *Store {
	return newStore(db, db)
}

func newStore(db *sql.DB, q DBTX) *Store {
	return &Store{
		db:          db,
		Admins:      &AdminRepository{q: q},
			Sessions:    &SessionRepository{q: q},
		Users:       &UserRepository{q: q},
		DeviceSlots: &DeviceSlotRepository{q: q},
		Renewals:    &RenewalRepository{q: q},
		System:      &SystemConfigRepository{q: q},
	}
}

func (s *Store) WithTx(ctx context.Context, fn func(*Store) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txStore := newStore(s.db, tx)
	if err := fn(txStore); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
