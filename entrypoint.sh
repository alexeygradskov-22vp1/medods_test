#!/bin/sh

until pg_isready -h $PG_HOST -p $PG_PORT -U $PG_USER; do
  sleep 1
done

goose -dir ./migrations postgres "host=$PG_HOST port=$PG_PORT user=$PG_USER password=$PG_PASSWORD dbname=$PG_DB sslmode=disable" up

./server
