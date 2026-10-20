package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Post is written by one user. The foreign key column is owned by the edge.
type Post struct{ ent.Schema }

func (Post) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
		field.String("slug").
			Unique(),
		field.Enum("status").
			Values("draft", "published").
			Default("draft"),
		field.Text("body").
			Optional(),
	}
}

func (Post) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("author", User.Type).
			Ref("posts").
			Unique().
			Required(),
		edge.To("comments", Comment.Type),
	}
}
