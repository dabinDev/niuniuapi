-- 创作产出记录:每次封面/拆书/剧本生成落一行,供「我的作品」回看。
-- output 用 jsonb 存结果(封面 data URI 可能较大)。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

CREATE TABLE IF NOT EXISTS creation_tasks (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL,
    type       VARCHAR(32) NOT NULL,
    title      VARCHAR(200) NOT NULL DEFAULT '',
    input      JSONB,
    output     JSONB,
    model      VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS creationtask_user_created ON creation_tasks (user_id, created_at);
CREATE INDEX IF NOT EXISTS creationtask_user_type ON creation_tasks (user_id, type);
