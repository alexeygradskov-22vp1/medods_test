package user

import (
	"context"
	"database/sql"
	"fmt"
	"medods/database"
	"medods/internal/repository"
	"strings"
)

type FieldName string

const (
	GuidFieldName FieldName = "guid"
)

type UserRepository struct {
	db     *database.Database
	txOpts *sql.TxOptions
}

func NewRepository(db *database.Database) *UserRepository {
	return &UserRepository{db: db, txOpts: nil}
}

func create(tx *sql.Tx, model *User) error {
	_, err := tx.Exec("INSERT INTO users (guid) values ($1)", model.Guid)
	return err
}

func (r *UserRepository) Create(ctx context.Context, model *User) error {
	return repository.TxRunner(ctx, r.db, func(tx *sql.Tx) error {
		return create(tx, model)
	})
}

func getOneBy(ctx context.Context, tx *sql.Tx, fields []FieldName, values []interface{}) (*User, error) {
	var user = new(User)
	if len(fields) < 1 || len(values) < 1 {
		return nil, fmt.Errorf("len of values and fields must be not zero")
	}

	if len(fields) != len(values) {
		return nil, fmt.Errorf("len of arguments is mismatch")
	}

	args := make([]string, len(fields))
	for i, v := range fields {
		args[i] = fmt.Sprintf("%s=$%d", v, i+1)
	}

	whereArgs := strings.Join(args, " AND ")

	q := fmt.Sprintf("SELECT * FROM users WHERE %s LIMIT 1", whereArgs)

	row := tx.QueryRowContext(ctx, q, values...)
	err := row.Scan(&user.Guid)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetOneBy(ctx context.Context, fields []FieldName, values []interface{}) (*User, error) {
	val, err := repository.GetTxRunner(ctx, r.db, func(tx *sql.Tx) (interface{}, error) {
		return getOneBy(ctx, tx, fields, values)
	})
	if err != nil {
		return nil, err
	}
	res, ok := val.(*User)
	if !ok {
		return nil, fmt.Errorf("get one by err: parse interface{} to *Token")
	}
	return res, nil
}
