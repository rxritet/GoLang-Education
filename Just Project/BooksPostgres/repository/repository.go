// Package repository реализует слой доступа к данным через PostgreSQL (pgx/v5).
//
// Интерфейс Repository абстрагирует хранилище;
// PostgresRepo — конкретная реализация поверх pgxpool.Pool.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"bookspostgres/models"
)

// Repository описывает операции с книгами.
type Repository interface {
	Create(ctx context.Context, b *models.Book) error
	GetAll(ctx context.Context) ([]*models.Book, error)
	GetByID(ctx context.Context, id string) (*models.Book, error)
	Update(ctx context.Context, b *models.Book) error
	Delete(ctx context.Context, id string) error
}

// PostgresRepo реализует Repository поверх pgxpool.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// New создаёт PostgresRepo с переданным пулом соединений.
func New(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// Create вставляет новую книгу и заполняет ID и CreatedAt из БД.
func (r *PostgresRepo) Create(ctx context.Context, b *models.Book) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO books (title, author, year)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		b.Title, b.Author, b.Year,
	)
	return row.Scan(&b.ID, &b.CreatedAt)
}

// GetAll возвращает все книги, отсортированные по дате создания (новые первыми).
func (r *PostgresRepo) GetAll(ctx context.Context) ([]*models.Book, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, author, year, created_at
		 FROM books
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.GetAll: %w", err)
	}
	defer rows.Close()

	var books []*models.Book
	for rows.Next() {
		b := &models.Book{}
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Year, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository.GetAll scan: %w", err)
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// GetByID возвращает книгу по UUID. Возвращает ошибку, если не найдена.
func (r *PostgresRepo) GetByID(ctx context.Context, id string) (*models.Book, error) {
	b := &models.Book{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, author, year, created_at
		 FROM books WHERE id = $1`,
		id,
	).Scan(&b.ID, &b.Title, &b.Author, &b.Year, &b.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}
	return b, nil
}

// Update обновляет метаданные книги. Возвращает ошибку, если строка не найдена.
func (r *PostgresRepo) Update(ctx context.Context, b *models.Book) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE books SET title=$1, author=$2, year=$3 WHERE id=$4`,
		b.Title, b.Author, b.Year, b.ID,
	)
	if err != nil {
		return fmt.Errorf("repository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("repository.Update: book %q not found", b.ID)
	}
	return nil
}

// Delete удаляет книгу по UUID.
func (r *PostgresRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM books WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("repository.Delete: book %q not found", id)
	}
	return nil
}
