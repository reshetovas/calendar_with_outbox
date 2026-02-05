package db

import (
	"calendar/pkg/config"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose"
)

type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Close()
}

type Postgres struct {
	Pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, conf config.Postgres) (*Postgres, error) {
	poolCfg, err := pgxpool.ParseConfig(conf.ConnString)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	// Устанавливаем минимальное количество соединений
	if conf.MaxConnections <= 0 {
		poolCfg.MaxConns = 5
	} else {
		poolCfg.MaxConns = conf.MaxConnections
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	// Миграции через database/sql на базе pgx stdlib.
	sqlDB := stdlib.OpenDB(*poolCfg.ConnConfig)
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		pool.Close()
		return nil, fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "resources/migrations"); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return &Postgres{Pool: pool}, nil
}

// ===== Транзакции через context =====

type txKey struct{}

func (p *Postgres) InjectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func (p *Postgres) ExtractTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// ===== Универсальные врапперы: если есть tx в контексте — используем его =====

func (p *Postgres) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if tx := p.ExtractTx(ctx); tx != nil {
		return tx.Exec(ctx, query, args...)
	}
	return p.Pool.Exec(ctx, query, args...)
}

func (p *Postgres) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if tx := p.ExtractTx(ctx); tx != nil {
		return tx.Query(ctx, query, args...)
	}
	return p.Pool.Query(ctx, query, args...)
}

func (p *Postgres) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if tx := p.ExtractTx(ctx); tx != nil {
		return tx.QueryRow(ctx, query, args...)
	}
	return p.Pool.QueryRow(ctx, query, args...)
}

// ===== Обёртка транзакции =====
// Коммит/роллбэк управляется единственным defer с именованным возвратом err.
func (p *Postgres) WithinTransaction(ctx context.Context, tFunc func(ctx context.Context) error) (err error) {
	tx, err := p.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		err = tx.Commit(ctx)
	}()

	// передаём вниз ctx с tx; все p.Exec/Query/QueryRow будут идти через этот tx
	err = tFunc(p.InjectTx(ctx, tx))
	return
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
