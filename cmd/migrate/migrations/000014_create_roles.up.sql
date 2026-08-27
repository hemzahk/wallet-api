CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT
);

INSERT INTO 
    roles (name, description)
VALUES
    (
        'customer',
        'A customer can topup, transfer, pay merchants and withdraw money'
    );

INSERT INTO 
    roles (name, description)
VALUES
    (
        'merchant',
        'A merchant can start a checkout session'
    );