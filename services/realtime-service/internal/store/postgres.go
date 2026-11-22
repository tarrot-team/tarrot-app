package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
)

const createMessagesTableSQL = `
CREATE TABLE IF NOT EXISTS chat_messages (
    id           TEXT PRIMARY KEY,
    room         VARCHAR(64) NOT NULL,
    from_user_id TEXT NOT NULL,
    text         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_room_created
    ON chat_messages (room, created_at DESC);
`

type PostgresStore struct {
	db *sql.DB
}

func OpenPostgres(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) EnsureSchema(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, createMessagesTableSQL); err != nil {
		return fmt.Errorf("ensure chat_messages schema: %w", err)
	}
	return nil
}

func (s *PostgresStore) SaveMessage(ctx context.Context, ev hub.Event) error {
	const query = `
		INSERT INTO chat_messages (id, room, from_user_id, text, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := s.db.ExecContext(ctx, query, ev.ID, ev.Room, ev.FromUserID, ev.Text, ev.CreatedAt); err != nil {
		return fmt.Errorf("insert chat message: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListMessages(ctx context.Context, room string, limit int) ([]hub.Event, error) {
	if limit <= 0 {
		limit = 20
	}
	const query = `
		SELECT id, room, from_user_id, text, created_at
		FROM chat_messages
		WHERE room = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, room, limit)
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	defer rows.Close()

	res := make([]hub.Event, 0)
	for rows.Next() {
		ev := hub.Event{Type: "message"}
		if err := rows.Scan(&ev.ID, &ev.Room, &ev.FromUserID, &ev.Text, &ev.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		res = append(res, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat messages: %w", err)
	}

	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return res, nil
}
