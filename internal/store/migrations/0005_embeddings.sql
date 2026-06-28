-- Semantic-search embeddings for spans (opt-in; populated on demand).
-- Vectors are stored as JSON arrays of floats; similarity is computed in Go,
-- so no DuckDB extension is required (fully local).

CREATE TABLE IF NOT EXISTS span_embeddings (
    project_id VARCHAR NOT NULL,
    trace_id   VARCHAR NOT NULL,
    span_id    VARCHAR NOT NULL,
    model      VARCHAR NOT NULL DEFAULT '',
    dim        INTEGER NOT NULL DEFAULT 0,
    vec        JSON    NOT NULL DEFAULT '[]',
    name       VARCHAR NOT NULL DEFAULT '',
    text       VARCHAR NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT now(),
    PRIMARY KEY (project_id, span_id)
);
