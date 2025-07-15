package blacklist

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/jmoiron/sqlx"
	"medods/database"
	"medods/internal/repository"
	"strings"
)

type FieldName string

const (
	AccessTokenFieldName FieldName = "access_token"
)

type BlacklistRepository struct {
	db     *database.Database
	txOpts *sql.TxOptions
}

func NewRepository(db *database.Database) *BlacklistRepository {
	return &BlacklistRepository{db: db}
}

func create(tx *sql.Tx, model *Blacklist) error {
	_, err := tx.Exec("INSERT INTO blacklists (access_token) values ($1)", model.AccessToken)
	return err
}

func (bl *BlacklistRepository) Create(ctx context.Context, model *Blacklist) error {
	return repository.TxRunner(ctx, bl.db, func(tx *sql.Tx) error {
		return create(tx, model)
	})
}

func getOneBy(ctx context.Context, tx *sql.Tx, fields []FieldName, values []interface{}) (*Blacklist, error) {
	var blacklist = new(Blacklist)
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

	q := fmt.Sprintf("SELECT * FROM blacklists WHERE %s", whereArgs)

	row := tx.QueryRowContext(ctx, q, values...)
	err := row.Scan(&blacklist.AccessToken)
	if err != nil {
		return nil, err
	}

	return blacklist, nil
}

func (r *BlacklistRepository) GetOneBy(ctx context.Context, fields []FieldName, values []interface{}) (*Blacklist, error) {
	val, err := repository.GetTxRunner(ctx, r.db, func(tx *sql.Tx) (interface{}, error) {
		return getOneBy(ctx, tx, fields, values)
	})
	if err != nil {
		return nil, err
	}
	res, ok := val.(*Blacklist)
	if !ok {
		return nil, fmt.Errorf("get one by err: parse interface{} to *Token")
	}
	return res, nil
}
