-- cc-otel schema (DuckDB). Single embedded analytical database.
-- Arrays/maps are stored as JSON text to stay within the purego driver's
-- scan capabilities (it does not scan native LIST/STRUCT values into Go types).

-- ---- Relational / tenancy tables -------------------------------------------

CREATE TABLE IF NOT EXISTS workspaces (
    id          VARCHAR PRIMARY KEY,
    name        VARCHAR NOT NULL,
    slug        VARCHAR NOT NULL UNIQUE,
    created_at  TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS projects (
    id             VARCHAR PRIMARY KEY,
    workspace_id   VARCHAR NOT NULL,
    name           VARCHAR NOT NULL,
    slug           VARCHAR NOT NULL,
    first_trace_at TIMESTAMP,
    created_at     TIMESTAMP DEFAULT now(),
    UNIQUE (workspace_id, slug)
);

CREATE TABLE IF NOT EXISTS api_keys (
    id            VARCHAR PRIMARY KEY,
    workspace_id  VARCHAR NOT NULL,
    name          VARCHAR,
    token_hash    VARCHAR NOT NULL UNIQUE,
    token_preview VARCHAR,
    created_at    TIMESTAMP DEFAULT now(),
    last_used_at  TIMESTAMP
);

-- ---- Telemetry: spans ------------------------------------------------------

CREATE TABLE IF NOT EXISTS spans (
    workspace_id           VARCHAR NOT NULL,
    project_id             VARCHAR NOT NULL,
    api_key_id             VARCHAR DEFAULT '',
    session_id             VARCHAR DEFAULT '',
    trace_id               VARCHAR NOT NULL,
    span_id                VARCHAR NOT NULL,
    parent_span_id         VARCHAR DEFAULT '',

    start_time_ns          BIGINT NOT NULL,
    end_time_ns            BIGINT NOT NULL,

    name                   VARCHAR DEFAULT '',
    service_name           VARCHAR DEFAULT '',
    kind                   TINYINT DEFAULT 0,
    status_code            SMALLINT DEFAULT 0,   -- 0=unset 1=ok 2=error
    status_message         VARCHAR DEFAULT '',
    error_type             VARCHAR DEFAULT '',
    scope_name             VARCHAR DEFAULT '',
    scope_version          VARCHAR DEFAULT '',

    -- GenAI / LLM enrichment
    operation              VARCHAR DEFAULT '',
    provider               VARCHAR DEFAULT '',
    model                  VARCHAR DEFAULT '',
    response_model         VARCHAR DEFAULT '',

    tokens_input           BIGINT DEFAULT 0,
    tokens_output          BIGINT DEFAULT 0,
    tokens_cache_read      BIGINT DEFAULT 0,
    tokens_cache_create    BIGINT DEFAULT 0,
    tokens_reasoning       BIGINT DEFAULT 0,

    cost_input_microcents  BIGINT DEFAULT 0,
    cost_output_microcents BIGINT DEFAULT 0,
    cost_total_microcents  BIGINT DEFAULT 0,
    cost_is_estimated      BOOLEAN DEFAULT FALSE,

    time_to_first_token_ns BIGINT DEFAULT 0,
    is_streaming           BOOLEAN DEFAULT FALSE,

    response_id            VARCHAR DEFAULT '',
    finish_reasons         JSON DEFAULT '[]',

    user_id                VARCHAR DEFAULT '',
    user_email             VARCHAR DEFAULT '',

    tool_call_id           VARCHAR DEFAULT '',
    tool_name              VARCHAR DEFAULT '',
    tool_input             VARCHAR DEFAULT '',
    tool_output            VARCHAR DEFAULT '',
    tool_names             JSON DEFAULT '[]',

    input_messages         JSON DEFAULT '[]',
    output_messages        JSON DEFAULT '[]',
    system_instructions    JSON DEFAULT '[]',
    tool_definitions       JSON DEFAULT '[]',
    events_json            JSON DEFAULT '[]',
    links_json             JSON DEFAULT '[]',

    tags                   JSON DEFAULT '[]',
    attributes             JSON DEFAULT '{}',
    resource               JSON DEFAULT '{}',

    ingested_at            TIMESTAMP DEFAULT now()
);

-- ---- Derived views: traces & sessions (aggregate-on-read) ------------------

CREATE OR REPLACE VIEW traces AS
SELECT
    workspace_id,
    project_id,
    trace_id,
    arg_max(session_id, CASE WHEN session_id <> '' THEN start_time_ns ELSE -1 END) AS session_id,
    count(*)                                                AS span_count,
    count(*) FILTER (WHERE status_code = 2)                 AS error_count,
    min(start_time_ns)                                      AS min_start_time_ns,
    max(end_time_ns)                                        AS max_end_time_ns,
    max(end_time_ns) - min(start_time_ns)                   AS duration_ns,
    sum(tokens_input)                                       AS tokens_input,
    sum(tokens_output)                                      AS tokens_output,
    sum(tokens_cache_read)                                  AS tokens_cache_read,
    sum(tokens_cache_create)                                AS tokens_cache_create,
    sum(tokens_reasoning)                                   AS tokens_reasoning,
    sum(cost_total_microcents)                              AS cost_total_microcents,
    coalesce(CAST(to_json(list(DISTINCT model) FILTER (WHERE model <> '')) AS VARCHAR), '[]')       AS models,
    coalesce(CAST(to_json(list(DISTINCT provider) FILTER (WHERE provider <> '')) AS VARCHAR), '[]') AS providers,
    arg_min(name, start_time_ns)                           AS root_span_name,
    arg_min(span_id, start_time_ns)                        AS root_span_id
FROM spans
GROUP BY workspace_id, project_id, trace_id;

CREATE OR REPLACE VIEW sessions AS
SELECT
    workspace_id,
    project_id,
    session_id,
    count(DISTINCT trace_id)                                AS trace_count,
    count(*)                                                AS span_count,
    count(*) FILTER (WHERE status_code = 2)                 AS error_count,
    min(start_time_ns)                                      AS min_start_time_ns,
    max(end_time_ns)                                        AS max_end_time_ns,
    max(end_time_ns) - min(start_time_ns)                   AS duration_ns,
    sum(tokens_input)                                       AS tokens_input,
    sum(tokens_output)                                      AS tokens_output,
    sum(tokens_cache_read)                                  AS tokens_cache_read,
    sum(tokens_cache_create)                                AS tokens_cache_create,
    sum(tokens_reasoning)                                   AS tokens_reasoning,
    sum(cost_total_microcents)                              AS cost_total_microcents,
    coalesce(CAST(to_json(list(DISTINCT model) FILTER (WHERE model <> '')) AS VARCHAR), '[]')       AS models,
    coalesce(CAST(to_json(list(DISTINCT provider) FILTER (WHERE provider <> '')) AS VARCHAR), '[]') AS providers,
    arg_max(user_id, CASE WHEN user_id <> '' THEN start_time_ns ELSE -1 END) AS user_id
FROM spans
WHERE session_id <> ''
GROUP BY workspace_id, project_id, session_id;
