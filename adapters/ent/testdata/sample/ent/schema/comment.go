package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Comment belongs to a post through an explicit edge field.
type Comment struct{ ent.Schema }

func (Comment) Fields() []ent.Field {
	return []ent.Field{
		field.String("text"),
		field.Int("post_id"),
	}
}

func (Comment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("post", Post.Type).
			Ref("comments").
			Field("post_id").
			Unique().
			Required(),
	}
}
