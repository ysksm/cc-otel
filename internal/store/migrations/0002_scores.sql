-- Scores (annotations + evaluation results) and evaluation definitions.

CREATE TABLE IF NOT EXISTS evaluations (
    id           VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL,
    project_id   VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,
    slug         VARCHAR NOT NULL,
    prompt       VARCHAR NOT NULL DEFAULT '',
    provider     VARCHAR NOT NULL DEFAULT '',
    model        VARCHAR NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP DEFAULT now(),
    UNIQUE (project_id, slug)
);

CREATE TABLE IF NOT EXISTS scores (
    id               VARCHAR PRIMARY KEY,
    workspace_id     VARCHAR NOT NULL,
    project_id       VARCHAR NOT NULL,
    trace_id         VARCHAR NOT NULL DEFAULT '',
    span_id          VARCHAR NOT NULL DEFAULT '',
    session_id       VARCHAR NOT NULL DEFAULT '',
    source           VARCHAR NOT NULL,            -- 'annotation' | 'evaluation'
    source_id        VARCHAR NOT NULL DEFAULT '', -- evaluation id, or 'UI'
    name             VARCHAR NOT NULL DEFAULT '', -- metric / evaluator name
    value            DOUBLE  NOT NULL DEFAULT 0,  -- normalized 0..1
    passed           BOOLEAN NOT NULL DEFAULT FALSE,
    errored          BOOLEAN NOT NULL DEFAULT FALSE,
    reasoning        VARCHAR NOT NULL DEFAULT '', -- judge explanation / annotation note
    duration_ns      BIGINT  NOT NULL DEFAULT 0,
    tokens           BIGINT  NOT NULL DEFAULT 0,
    cost_microcents  BIGINT  NOT NULL DEFAULT 0,
    created_at       TIMESTAMP DEFAULT now()
);

-- Per-trace score rollup for list badges.
CREATE OR REPLACE VIEW trace_scores AS
SELECT
    project_id,
    trace_id,
    count(*)                                  AS score_count,
    count(*) FILTER (WHERE passed)            AS passed_count,
    count(*) FILTER (WHERE NOT passed AND NOT errored) AS failed_count,
    avg(value)                                AS avg_value
FROM scores
WHERE trace_id <> ''
GROUP BY project_id, trace_id;
