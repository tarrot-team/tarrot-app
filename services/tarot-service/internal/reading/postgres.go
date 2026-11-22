package reading

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/spread"
)

const createReadingsTableSQL = `
CREATE TABLE IF NOT EXISTS readings (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL,
    mode           VARCHAR(16) NOT NULL,
    spread_type    VARCHAR(32) NOT NULL,
    spread_title   TEXT NOT NULL,
    question       TEXT NOT NULL DEFAULT '',
    cards_json     JSONB NOT NULL,
    interpretation TEXT NOT NULL,
    source         VARCHAR(16) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_readings_user_created
    ON readings (user_id, created_at DESC);
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
	if _, err := s.db.ExecContext(ctx, createReadingsTableSQL); err != nil {
		return fmt.Errorf("ensure readings schema: %w", err)
	}
	return nil
}

func (s *PostgresStore) Save(ctx context.Context, r Reading) error {
	cardsJSON, err := json.Marshal(r.Spread.Cards)
	if err != nil {
		return fmt.Errorf("marshal cards: %w", err)
	}

	const query = `
		INSERT INTO readings (
			id, user_id, mode, spread_type, spread_title, question, cards_json, interpretation, source, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = s.db.ExecContext(
		ctx, query,
		r.ID, r.UserID, r.Mode, r.Spread.Type, r.Spread.Title,
		r.Question, cardsJSON, r.Interpretation, r.Source, r.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert reading: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetByID(ctx context.Context, id string) (Reading, error) {
	const query = `
		SELECT id, user_id, mode, spread_type, spread_title, question, cards_json, interpretation, source, created_at
		FROM readings
		WHERE id = $1
	`
	var (
		r         Reading
		cardsJSON []byte
	)
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.UserID, &r.Mode, &r.Spread.Type, &r.Spread.Title,
		&r.Question, &cardsJSON, &r.Interpretation, &r.Source, &r.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Reading{}, ErrNotFound
		}
		return Reading{}, fmt.Errorf("select reading: %w", err)
	}
	if err := json.Unmarshal(cardsJSON, &r.Spread.Cards); err != nil {
		return Reading{}, fmt.Errorf("unmarshal cards: %w", err)
	}
	return r, nil
}

func (s *PostgresStore) ListByUser(ctx context.Context, userID string, limit int) ([]Reading, error) {
	if limit <= 0 {
		limit = 20
	}
	const query = `
		SELECT id, user_id, mode, spread_type, spread_title, question, cards_json, interpretation, source, created_at
		FROM readings
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list readings: %w", err)
	}
	defer rows.Close()

	res := make([]Reading, 0)
	for rows.Next() {
		var (
			r         Reading
			cardsJSON []byte
		)
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.Mode, &r.Spread.Type, &r.Spread.Title,
			&r.Question, &cardsJSON, &r.Interpretation, &r.Source, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reading: %w", err)
		}
		var cards []spread.PositionCard
		if err := json.Unmarshal(cardsJSON, &cards); err != nil {
			return nil, fmt.Errorf("unmarshal cards: %w", err)
		}
		r.Spread.Cards = cards
		res = append(res, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate readings: %w", err)
	}
	return res, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id, userID string) error {
	var ownerID string
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM readings WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("select reading owner: %w", err)
	}
	if ownerID != userID {
		return ErrForbidden
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM readings WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete reading: %w", err)
	}
	return nil
}
