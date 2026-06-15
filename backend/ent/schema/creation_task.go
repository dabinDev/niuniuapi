package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CreationTask 记录一次创作产出（封面 / 拆书 / 剧本），供「我的作品」回看。
//
// type: cover | teardown | script
// input/output 以 JSON 存请求与结果（结果可能较大，如封面 data URI）。
type CreationTask struct {
	ent.Schema
}

func (CreationTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "creation_tasks"},
	}
}

func (CreationTask) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("type").MaxLen(32),
		field.String("title").MaxLen(200).Default(""),
		field.Text("input").Optional().SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Text("output").Optional().SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("model").MaxLen(128).Default(""),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CreationTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
		index.Fields("user_id", "type"),
	}
}
