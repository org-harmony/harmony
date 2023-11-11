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
    created_by  UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    updated_at  TIMESTAMPTZ
);

CREATE TABLE templates
(
    id             UUID PRIMARY KEY,
    template_set   UUID         NOT NULL REFERENCES template_sets (id) ON DELETE CASCADE,
    type           VARCHAR(255) NOT NULL,
    name           VARCHAR(255) NOT NULL,
    version        VARCHAR(255) NOT NULL,
    json           JSONB        NOT NULL,
    created_by     UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    updated_at     TIMESTAMPTZ
);
