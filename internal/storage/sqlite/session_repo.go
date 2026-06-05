package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"4vpx/internal/domain"
)

type SessionRepository struct {
	q DBTX
}

func (r *SessionRepository) Create(ctx context.Context, session domain.AdminSession) error {
	_, err := r.q.ExecContext(ctx, `
        INSERT INTO admin_sessions (token, admin_id, created_at, updated_at)
        VALUES (?, ?, ?, ?)
    `, session.Token, session.AdminID, formatTime(session.CreatedAt), formatTime(session.UpdatedAt))
	if err != nil {
		return fmt.Errorf("insert admin session: %w", err)
	}
	return nil
}

func (r *SessionRepository) GetByToken(ctx context.Context, token string) (domain.AdminSession, error) {
	return scanAdminSession(r.q.QueryRowContext(ctx, `
        SELECT token, admin_id, created_at, updated_at
        FROM admin_sessions
        WHERE token = ?
    `, token))
}

func (r *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM admin_sessions WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("delete admin session: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
