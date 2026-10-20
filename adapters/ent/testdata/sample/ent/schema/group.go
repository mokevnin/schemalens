package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Group has many users; the join table is created by the edge.
type Group struct{ ent.Schema }

func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
	}
}

// Titles are unique only among non-empty ones: the model itself makes the index partial.
func (Group) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("title").
			Unique().
			Annotations(entsql.IndexWhere("title <> ''")),
	}
}

func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("members", User.Type).
			Ref("groups"),
	}
}
