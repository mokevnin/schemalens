package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User is a person who writes posts.
type User struct{ ent.Schema }

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{Timestamps{}}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(100),
		field.String("email").
			Unique(),
		field.Int("age").
			Optional(),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("posts", Post.Type),
		edge.To("groups", Group.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "age").
			Unique(),
	}
}
