-- +goose Up
-- +goose StatementBegin
create table if not exists blacklists(
    access_token varchar
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists blacklists;
-- +goose StatementEnd
