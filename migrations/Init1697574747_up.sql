CREATE TABLE users
(
    id         UUID PRIMARY KEY,
    email      VARCHAR(255) NOT NULL UNIQUE,
    firstname  VARCHAR(255) NOT NULL,
    lastname   VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ
);

CREATE TABLE sessions
(
    id         UUID PRIMARY KEY,
    type       VARCHAR(255) NOT NULL,
    payload    JSONB        NOT NULL,
    meta       JSONB,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    expires_at TIMESTAMPTZ  NOT NULL,
    updated_at TIMESTAMPTZ
);