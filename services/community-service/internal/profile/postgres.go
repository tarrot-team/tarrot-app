package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const createProfilesTableSQL = `
CREATE TABLE IF NOT EXISTS profiles (
    user_id     TEXT PRIMARY KEY,
    username    VARCHAR(32) NOT NULL,
    bio         TEXT NOT NULL DEFAULT '',
    avatar_url  TEXT NOT NULL DEFAULT '',
    birth_date  VARCHAR(10) NOT NULL DEFAULT '',
    level       INT NOT NULL DEFAULT 1,
    reputation  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_username_lower
    ON profiles (LOWER(username));
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
	if _, err := s.db.ExecContext(ctx, createProfilesTableSQL); err != nil {
		return fmt.Errorf("ensure profiles schema: %w", err)
	}
	return nil
}

func (s *PostgresStore) Create(ctx context.Context, p Profile) error {
	const query = `
		INSERT INTO profiles (
			user_id, username, bio, avatar_url, birth_date, level, reputation, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.ExecContext(
		ctx, query,
		p.UserID, p.Username, p.Bio, p.AvatarURL, p.BirthDate,
		p.Level, p.Reputation, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "profiles_pkey" {
				return ErrAlreadyExists
			}
			return ErrUsernameTaken
		}
		return fmt.Errorf("insert profile: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetByID(ctx context.Context, userID string) (Profile, error) {
	const query = `
		SELECT user_id, username, bio, avatar_url, birth_date, level, reputation, created_at, updated_at
		FROM profiles
		WHERE user_id = $1
	`
	var p Profile
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&p.UserID, &p.Username, &p.Bio, &p.AvatarURL, &p.BirthDate,
		&p.Level, &p.Reputation, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Profile{}, ErrNotFound
		}
		return Profile{}, fmt.Errorf("select profile: %w", err)
	}
	return p, nil
}

func (s *PostgresStore) Update(ctx context.Context, p Profile) error {
	const query = `
		UPDATE profiles
		SET username = $2,
		    bio = $3,
		    avatar_url = $4,
		    birth_date = $5,
		    level = $6,
		    reputation = $7,
		    updated_at = $8
		WHERE user_id = $1
	`
	res, err := s.db.ExecContext(
		ctx, query,
		p.UserID, p.Username, p.Bio, p.AvatarURL, p.BirthDate,
		p.Level, p.Reputation, p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUsernameTaken
		}
		return fmt.Errorf("update profile: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
