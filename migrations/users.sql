CREATE TABLE IF NOT EXISTS gofire_schema.users
(
    id       SERIAL PRIMARY KEY,
    username varchar(100),
    password varchar(255)
);
