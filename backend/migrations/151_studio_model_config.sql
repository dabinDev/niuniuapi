-- 创作模型配置:每用户一行,记录其"生图模型 / 文案模型"(哪把密钥 + 哪个模型),
-- 以 jsonb 存 {image:{api_key_id,model}, text:{api_key_id,model}}。
-- 供「API 密钥」页配置、创作功能(封面/拆书/剧本)读取。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

CREATE TABLE IF NOT EXISTS studio_model_configs (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL,
    config     JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS studiomodelconfig_user_id ON studio_model_configs (user_id);
