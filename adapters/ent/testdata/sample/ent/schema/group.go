package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Group has many users; the join table is created by the edge.
type Group struct{ ent.Schema }

func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
	}
}

func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("members", User.Type).
			Ref("groups"),
	}
}
