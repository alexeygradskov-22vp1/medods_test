package repository

import (
	"context"
	"database/sql"
	"fmt"
	"medods/database"
)

type TxFunc func(tx *sql.Tx) error
type GetTxFunc func(tx *sql.Tx) (interface{}, error)

type GetManyTxFunc func(tx *sql.Tx) ([]interface{}, error)

func TxRunner(ctx context.Context, db *database.Database, txFunc TxFunc) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	err = txFunc(tx)
	return err
}

func GetTxRunner(ctx context.Context, db *database.Database, txFunc GetTxFunc) (interface{}, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	val, err := txFunc(tx)
	return val, err
}

func GetManyTxRunner(ctx context.Context, db *database.Database, txFunc GetManyTxFunc) ([]interface{}, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	val, err := txFunc(tx)
	return val, err
}
