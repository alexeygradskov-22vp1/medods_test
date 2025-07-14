package uow

import (
	"context"
	"database/sql"
	"fmt"
	"medods/internal/repository/blacklist"
	"medods/internal/repository/token"
	"medods/internal/repository/user"
	"runtime"
)

type UnitOfWork struct {
	db        *sql.DB
	User      *user.CommandRepository
	Token     *token.CommandRepository
	BlackList *blacklist.CommandRepository
}

func NewUnitOfWork(db *sql.DB) *UnitOfWork {
	return &UnitOfWork{
		db:        db,
		User:      user.NewCommand(db, user.DollarWildcard),
		Token:     token.NewCommand(db, token.DollarWildcard),
		BlackList: blacklist.NewCommand(db, blacklist.DollarWildcard),
	}
}

func (u *UnitOfWork) WithTransaction(ctx context.Context, fn func(uow *UnitOfWork) error) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			switch e := p.(type) {
			case runtime.Error:
				err = fmt.Errorf("error occurred while processing transaction: %v", p)
				return
			case error:
				err = fmt.Errorf("error occurred while processing transaction: %v", p)
				return
			default:
				panic(e)
			}
		}
		if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}

	}()

	txUow := &UnitOfWork{
		User:      user.NewCommand(tx, user.DollarWildcard),
		Token:     token.NewCommand(tx, token.DollarWildcard),
		BlackList: blacklist.NewCommand(tx, blacklist.DollarWildcard),
	}
	return fn(txUow)
}
