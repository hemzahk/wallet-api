CREATE TABLE IF NOT EXISTS user_invitations (
    token BYTEA PRIMARY KEY,
    user_id UUID NOT NULL,
    expiry TIMESTAMP NOT NULL
);