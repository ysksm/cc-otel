-- Monitors (threshold rules) and the signals they generate.

CREATE TABLE IF NOT EXISTS monitors (
    id             VARCHAR PRIMARY KEY,
    workspace_id   VARCHAR NOT NULL,
    project_id     VARCHAR NOT NULL,
    name           VARCHAR NOT NULL,
    condition_type VARCHAR NOT NULL,            -- trace_error | cost_gt | latency_gt | tokens_gt | model_used
    threshold      DOUBLE  NOT NULL DEFAULT 0,  -- USD for cost_gt, ms for latency_gt, count for tokens_gt
    value_str      VARCHAR NOT NULL DEFAULT '', -- model name for model_used
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMP DEFAULT now(),
    UNIQUE (project_id, name)
);

CREATE TABLE IF NOT EXISTS signals (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    project_id   VARCHAR NOT NULL,
    monitor_id   VARCHAR NOT NULL,
    monitor_name VARCHAR NOT NULL DEFAULT '',
    trace_id     VARCHAR NOT NULL DEFAULT '',
    session_id   VARCHAR NOT NULL DEFAULT '',
    description  VARCHAR NOT NULL DEFAULT '',
    value        DOUBLE  NOT NULL DEFAULT 0,
    created_at   TIMESTAMP DEFAULT now()
);
