CREATE TABLE IF NOT EXISTS metrics(
    id VARCHAR(50),
    mtype VARCHAR(10),
    delta BIGINT,
    value DOUBLE PRECISION,
    hash VARCHAR(100),
    PRIMARY KEY(id, mtype)
);
