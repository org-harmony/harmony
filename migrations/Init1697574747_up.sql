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

CREATE TABLE template_sets
(
    id          UUID PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    version     VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    updated_at  TIMESTAMPTZ
);

CREATE TABLE templates
(
    id             UUID PRIMARY KEY,
    template_type  VARCHAR(255) NOT NULL,
    template_set   UUID         NOT NULL REFERENCES template_sets (id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    template       JSONB        NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    updated_at     TIMESTAMPTZ
);
