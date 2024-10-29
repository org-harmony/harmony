#!/bin/bash
set -e

if [ "$RUN_MIGRATIONS" = "true" ]; then
    # TODO simplify, to speed up
    echo "Building migration tool..."
    go build -o migrate src/cmd/migrate/*.go

    echo "Running migrations..."
    ./migrate up

    echo "Removing migration tool..."
    rm ./migrate
fi

echo "Starting development server..."
exec air