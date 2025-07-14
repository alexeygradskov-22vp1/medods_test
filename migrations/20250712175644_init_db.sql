-- +goose Up
-- +goose StatementBegin
create table if not exists users(
    guid varchar primary key
);
create table if not exists tokens(
    id bigint generated always as identity primary key,
    user_guid varchar references users(guid),
    refresh_token varchar,
    active bool,
    user_agent varchar,
    ip varchar
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists tokens;
drop table if exists users;
-- +goose StatementEnd
