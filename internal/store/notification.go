package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Notification = notifikasi in-app untuk user.
type Notification struct {
	ID        string
	UserID    string
	Type      string // "disposisi_baru" | "komentar_baru"
	Payload   json.RawMessage
	ReadAt    *time.Time
	CreatedAt time.Time
}

// NotificationInput parameter untuk insert notifikasi.
type NotificationInput struct {
	ID      string // UUIDv7 caller-generated
	UserID  string
	Type    string
	Payload json.RawMessage
}

// CreateNotification insert tunggal — typically dipanggil dari trigger handler
// (after CreateDisposisi / AppendKomentar). Self-notification di-skip:
// kalau user_id == actor (set caller-side), tidak buat notifikasi.
func (s *Store) CreateNotification(ctx context.Context, in NotificationInput) error {
	const q = `
		INSERT INTO notifications (id, user_id, type, payload_jsonb)
		VALUES ($1, $2, $3, $4)`
	_, err := s.pool.Exec(ctx, q, in.ID, in.UserID, in.Type, in.Payload)
	if err != nil {
		return fmt.Errorf("store: insert notif: %w", err)
	}
	return nil
}

// CreateNotificationTx variant untuk dipakai di dalam transaction caller
// (mis. saat AppendKomentar yang sudah punya tx — extend secara atomic).
// Caller bertanggungjawab manage tx.
func (s *Store) CreateNotificationTx(ctx context.Context, tx pgx.Tx, in NotificationInput) error {
	const q = `
		INSERT INTO notifications (id, user_id, type, payload_jsonb)
		VALUES ($1, $2, $3, $4)`
	_, err := tx.Exec(ctx, q, in.ID, in.UserID, in.Type, in.Payload)
	if err != nil {
		return fmt.Errorf("store: insert notif tx: %w", err)
	}
	return nil
}

// ListNotificationsFilter filter untuk query.
type ListNotificationsFilter struct {
	UserID     string
	UnreadOnly bool
	Limit      int
}

// ListNotifications return notifikasi untuk user, sorted by created_at DESC.
// Menggunakan partial index idx_notif_user_unread saat UnreadOnly=true.
func (s *Store) ListNotifications(ctx context.Context, f ListNotificationsFilter) ([]Notification, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}

	q := `SELECT id::text, user_id::text, type, payload_jsonb, read_at, created_at
		FROM notifications
		WHERE user_id = $1`
	args := []interface{}{f.UserID}
	if f.UnreadOnly {
		q += ` AND read_at IS NULL`
	}
	q += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d`, f.Limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list notif: %w", err)
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Payload, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan notif: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// CountUnread untuk badge.
func (s *Store) CountUnreadNotifications(ctx context.Context, userID string) (int, error) {
	const q = `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`
	var count int
	err := s.pool.QueryRow(ctx, q, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: count unread: %w", err)
	}
	return count, nil
}

// MarkNotificationRead set read_at=NOW() — idempotent (already read = no-op).
func (s *Store) MarkNotificationRead(ctx context.Context, id, userID string) error {
	const q = `UPDATE notifications SET read_at = NOW()
		WHERE id = $1 AND user_id = $2 AND read_at IS NULL`
	_, err := s.pool.Exec(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("store: mark read: %w", err)
	}
	return nil
}

// MarkAllNotificationsRead set read_at=NOW() untuk semua unread user.
func (s *Store) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	const q = `UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND read_at IS NULL`
	_, err := s.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("store: mark all read: %w", err)
	}
	return nil
}
