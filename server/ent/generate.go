//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --feature privacy,entql,namedges,sql/upsert,sql/execquery,intercept ./schema
package ent
