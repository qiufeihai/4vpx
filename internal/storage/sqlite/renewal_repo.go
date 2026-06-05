package sqlite

import (
	"context"
	"fmt"

	"4vpx/internal/domain"
)

type RenewalRepository struct {
	q DBTX
}

func (r *RenewalRepository) List(ctx context.Context) ([]domain.RenewalRecord, error) {
	rows, err := r.q.QueryContext(ctx, `
        SELECT id, user_id, days, action, before_expires_at, after_expires_at, notes, actor, created_at
        FROM renewal_records
        ORDER BY created_at DESC, id DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("list renewals: %w", err)
	}
	defer rows.Close()

	records := make([]domain.RenewalRecord, 0)
	for rows.Next() {
		record, err := scanRenewalRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate renewals: %w", err)
	}
	return records, nil
}

func (r *RenewalRepository) Create(ctx context.Context, record domain.RenewalRecord) (domain.RenewalRecord, error) {
	res, err := r.q.ExecContext(ctx, `
        INSERT INTO renewal_records (
            user_id, days, action, before_expires_at, after_expires_at, notes, actor, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `,
		record.UserID,
		record.Days,
		record.Action,
		formatTime(record.BeforeExpiresAt),
		formatTime(record.AfterExpiresAt),
		record.Notes,
		record.Actor,
		formatTime(record.CreatedAt),
	)
	if err != nil {
		return domain.RenewalRecord{}, fmt.Errorf("insert renewal record: %w", err)
	}

	record.ID, err = res.LastInsertId()
	if err != nil {
		return domain.RenewalRecord{}, fmt.Errorf("renewal last insert id: %w", err)
	}
	return record, nil
}

func (r *RenewalRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.RenewalRecord, error) {
	rows, err := r.q.QueryContext(ctx, `
        SELECT id, user_id, days, action, before_expires_at, after_expires_at, notes, actor, created_at
        FROM renewal_records
        WHERE user_id = ?
        ORDER BY created_at DESC, id DESC
    `, userID)
	if err != nil {
		return nil, fmt.Errorf("list renewals: %w", err)
	}
	defer rows.Close()

	records := make([]domain.RenewalRecord, 0)
	for rows.Next() {
		record, err := scanRenewalRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate renewals: %w", err)
	}
	return records, nil
}
