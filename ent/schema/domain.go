package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Domain struct {
	ent.Schema
}

func (Domain) Fields() []ent.Field {
	return []ent.Field{
		field.String("landing_domain").
			Comment("Unique landing domain"),
		field.Time("created_at").
			Default(time.Now).
			Comment("When this domain was discovered"),
	}
}

func (Domain) Edges() []ent.Edge {
	return nil
}

func (Domain) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("landing_domain").Unique(),
	}
}
