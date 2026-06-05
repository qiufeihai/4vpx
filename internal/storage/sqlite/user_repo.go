package sqlite

import (
	"context"
	"fmt"
	"strings"

	"4vpx/internal/domain"
)

type UserRepository struct {
	q DBTX
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	res, err := r.q.ExecContext(ctx, `
        INSERT INTO users (
            name, notes, enabled, expires_at, access_token, device_slots, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `,
		strings.TrimSpace(user.Name),
		user.Notes,
		boolToInt(user.Enabled),
		formatTime(user.ExpiresAt),
		user.AccessToken,
		user.DeviceSlots,
		formatTime(user.CreatedAt),
		formatTime(user.UpdatedAt),
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}

	user.ID, err = res.LastInsertId()
	if err != nil {
		return domain.User{}, fmt.Errorf("user last insert id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user domain.User) error {
	res, err := r.q.ExecContext(ctx, `
        UPDATE users
        SET name = ?, notes = ?, enabled = ?, expires_at = ?, access_token = ?, device_slots = ?, updated_at = ?
        WHERE id = ?
    `,
		strings.TrimSpace(user.Name),
		user.Notes,
		boolToInt(user.Enabled),
		formatTime(user.ExpiresAt),
		user.AccessToken,
		user.DeviceSlots,
		formatTime(user.UpdatedAt),
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return ensureRowsAffected(res)
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	return scanUser(r.q.QueryRowContext(ctx, `
        SELECT id, name, notes, enabled, expires_at, access_token, device_slots, created_at, updated_at
        FROM users
        WHERE id = ?
    `, id))
}

func (r *UserRepository) GetByAccessToken(ctx context.Context, token string) (domain.User, error) {
	return scanUser(r.q.QueryRowContext(ctx, `
        SELECT id, name, notes, enabled, expires_at, access_token, device_slots, created_at, updated_at
        FROM users
        WHERE access_token = ?
    `, strings.TrimSpace(token)))
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.q.QueryContext(ctx, `
        SELECT id, name, notes, enabled, expires_at, access_token, device_slots, created_at, updated_at
        FROM users
        ORDER BY created_at DESC, id DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return ensureRowsAffected(res)
}
