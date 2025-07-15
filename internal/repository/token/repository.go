package token

import (
	"context"
	"database/sql"
	"fmt"
	"medods/database"
	"medods/internal/repository"
	"strings"
)

type TokenRepository struct {
	db     *database.Database
	txOpts *sql.TxOptions
}

type FieldName string

const (
	IdField           FieldName = "id"
	UserGuidField     FieldName = "user_guid"
	RefreshTokenField FieldName = "refresh_token"
	ActiveField       FieldName = "active"
	UserAgentField    FieldName = "user_agent"
	IpField           FieldName = "ip"
)

func NewRepository(db *database.Database) *TokenRepository {
	return &TokenRepository{db: db}
}

func create(tx *sql.Tx, model *Token) error {
	_, err := tx.Exec("INSERT INTO tokens (user_guid,refresh_token,active,user_agent,ip) values ($1,$2,$3,$4,$5)",
		model.UserGuid, model.RefreshToken, model.Active, model.UserAgent, model.IP)
	return err
}

func (r *TokenRepository) Create(ctx context.Context, model *Token) error {
	return repository.TxRunner(ctx, r.db, func(tx *sql.Tx) error {
		return create(tx, model)
	})
}

func getOneBy(ctx context.Context, tx *sql.Tx, fields []FieldName, values []interface{}) (*Token, error) {
	var token = new(Token)
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

	q := fmt.Sprintf("SELECT * FROM tokens WHERE %s LIMIT 1", whereArgs)

	row := tx.QueryRowContext(ctx, q, values...)
	err := row.Scan(&token.ID, &token.UserGuid, &token.RefreshToken, &token.Active, &token.UserAgent, &token.IP)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (r *TokenRepository) GetOneBy(ctx context.Context, fields []FieldName, values []interface{}) (*Token, error) {
	val, err := repository.GetTxRunner(ctx, r.db, func(tx *sql.Tx) (interface{}, error) {
		return getOneBy(ctx, tx, fields, values)
	})
	if err != nil {
		return nil, err
	}
	res, ok := val.(*Token)
	if !ok {
		return nil, fmt.Errorf("get one by err: parse interface{} to *Token")
	}
	return res, nil
}

func update(ctx context.Context, updated *Token, tx *sql.Tx, whereFields []FieldName, whereValues []interface{}) error {
	if len(whereFields) < 1 || len(whereValues) < 1 {
		return fmt.Errorf("len of values and fields must be not zero")
	}
	if len(whereFields) != len(whereValues) {
		return fmt.Errorf("len of arguments is mismatch")
	}

	//aggregate values for update
	updatedValues := []interface{}{
		updated.UserGuid,
		updated.RefreshToken,
		updated.Active,
		updated.UserAgent,
		updated.IP,
	}

	var numOfFields = 5

	args := make([]string, len(whereFields))
	for i, v := range whereFields {
		args[i] = fmt.Sprintf("%s=$%d", v, numOfFields+i+1)
	}

	whereArgs := strings.Join(args, " AND ")
	whereStr := fmt.Sprintf("WHERE %s", whereArgs)

	//aggregate all arguments for query
	allValues := append(updatedValues, whereValues...)

	//build sql query
	q := fmt.Sprintf("UPDATE tokens SET user_guid = $1, refresh_token = $2, active = $3, user_agent=$4, ip= $5 %s", whereStr)

	//execute sql query
	_, err := tx.ExecContext(ctx, q, allValues...)
	if err != nil {
		return err
	}

	return nil
}

func (r *TokenRepository) Update(ctx context.Context, updated *Token, whereFields []FieldName, whereValues []interface{}) error {
	return repository.TxRunner(ctx, r.db, func(tx *sql.Tx) error {
		return update(ctx, updated, tx, whereFields, whereValues)
	})
}

func getManyBy(ctx context.Context, tx *sql.Tx, fields []FieldName, values []interface{}) ([]Token, error) {
	tokens := make([]Token, 0)

	if len(fields) < 1 || len(values) < 1 {
		return nil, fmt.Errorf("len of values and fields must be not zero")
	}

	if len(fields) != len(values) {
		return nil, fmt.Errorf("len of arguments is mismatch")
	}
	//build where conditions
	args := make([]string, len(fields))
	for i, v := range fields {
		args[i] = fmt.Sprintf("%s=$%d", v, i+1)
	}

	whereArgs := strings.Join(args, " AND ")

	q := fmt.Sprintf("SELECT * FROM tokens WHERE %s", whereArgs)

	rows, err := tx.QueryContext(ctx, q, values...)
	if err != nil {
		return tokens, fmt.Errorf("get tokens error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var token Token
		err = rows.Scan(
			&token.ID,
			&token.UserGuid,
			&token.RefreshToken,
			&token.Active,
			&token.UserAgent,
			&token.IP)
		if err != nil {
			return nil, fmt.Errorf("scan tokens error: %w", err)
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (r *TokenRepository) GetManyBy(ctx context.Context, fields []FieldName, values []interface{}) (tokens []Token, err error) {
	vals, err := repository.GetManyTxRunner(ctx, r.db, func(tx *sql.Tx) ([]interface{}, error) {

		rawTokens, err := getManyBy(ctx, tx, fields, values)
		if err != nil {
			return nil, err
		}

		data := make([]interface{}, len(rawTokens))

		for i, v := range rawTokens {
			data[i] = v
		}
		return data, err
	})

	if err != nil {
		return nil, fmt.Errorf("GetManyBy: %w", err)
	}

	tokens = make([]Token, len(vals))
	for i, v := range vals {
		f, ok := v.(Token)
		if !ok {
			return nil, fmt.Errorf("convert interface{} to Token struct err")
		}
		tokens[i] = f
	}
	return tokens, nil
}
