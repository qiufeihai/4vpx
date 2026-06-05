package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"4vpx/internal/domain"
)

type AdminRepository struct {
	q DBTX
}

func (r *AdminRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count)
	return count, err
}

func (r *AdminRepository) List(ctx context.Context) ([]domain.Admin, error) {
	rows, err := r.q.QueryContext(ctx, `
        SELECT id, username, password_hash, created_at, updated_at
        FROM admins
        ORDER BY id ASC
    `)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	defer rows.Close()

	admins := make([]domain.Admin, 0)
	for rows.Next() {
		admin, err := scanAdmin(rows)
		if err != nil {
			return nil, err
		}
		admins = append(admins, admin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admins: %w", err)
	}
	return admins, nil
}

func (r *AdminRepository) Create(ctx context.Context, admin domain.Admin) (domain.Admin, error) {
	res, err := r.q.ExecContext(ctx, `
        INSERT INTO admins (username, password_hash, created_at, updated_at)
        VALUES (?, ?, ?, ?)
    `, strings.TrimSpace(admin.Username), admin.PasswordHash, formatTime(admin.CreatedAt), formatTime(admin.UpdatedAt))
	if err != nil {
		return domain.Admin{}, fmt.Errorf("insert admin: %w", err)
	}

	admin.ID, err = res.LastInsertId()
	if err != nil {
		return domain.Admin{}, fmt.Errorf("admin last insert id: %w", err)
	}
	return admin, nil
}

func (r *AdminRepository) GetByID(ctx context.Context, id int64) (domain.Admin, error) {
	return scanAdmin(r.q.QueryRowContext(ctx, `
        SELECT id, username, password_hash, created_at, updated_at
        FROM admins
        WHERE id = ?
    `, id))
}

func (r *AdminRepository) GetByUsername(ctx context.Context, username string) (domain.Admin, error) {
	return scanAdmin(r.q.QueryRowContext(ctx, `
        SELECT id, username, password_hash, created_at, updated_at
        FROM admins
        WHERE username = ?
    `, strings.TrimSpace(username)))
}

func (r *AdminRepository) UpdatePasswordHash(ctx context.Context, id int64, passwordHash string, updatedAt time.Time) error {
	res, err := r.q.ExecContext(ctx, `
        UPDATE admins
        SET password_hash = ?, updated_at = ?
        WHERE id = ?
    `, passwordHash, formatTime(updatedAt), id)
	if err != nil {
		return fmt.Errorf("update admin password: %w", err)
	}
	return ensureRowsAffected(res)
}

func (r *AdminRepository) UpdateCredentials(ctx context.Context, id int64, username, passwordHash string, updatedAt time.Time) error {
	res, err := r.q.ExecContext(ctx, `
        UPDATE admins
        SET username = ?, password_hash = ?, updated_at = ?
        WHERE id = ?
    `, strings.TrimSpace(username), passwordHash, formatTime(updatedAt), id)
	if err != nil {
		return fmt.Errorf("update admin credentials: %w", err)
	}
	return ensureRowsAffected(res)
}
