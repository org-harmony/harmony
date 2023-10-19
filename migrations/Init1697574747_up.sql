CREATE TABLE users
(
    id       UUID PRIMARY KEY,
    email      VARCHAR(255) NOT NULL UNIQUE,
    firstname  VARCHAR(255) NOT NULL,
    lastname   VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ
);
