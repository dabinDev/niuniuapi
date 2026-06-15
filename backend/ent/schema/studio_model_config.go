package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// StudioModelConfig 保存用户的创作模型配置（生图 / 文案模型）。
//
// 每用户一行（user_id 唯一），config 以 JSON 存
// {"image":{"api_key_id":1,"model":"gpt-image-1"},"text":{...}}。
type StudioModelConfig struct {
	ent.Schema
}

func (StudioModelConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "studio_model_configs"},
	}
}

func (StudioModelConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Unique(),
		field.Text("config").
			Default("{}").
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}
