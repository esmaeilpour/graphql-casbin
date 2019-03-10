package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/casbin/casbin"
	"github.com/graphql-go/graphql"
)

var rootQuery = graphql.NewObject(graphql.ObjectConfig{
	Name: "RootQuery",
	Fields: graphql.Fields{
		"foo": &graphql.Field{
			Type: graphql.String,
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				return "bar", nil
			},
		},
	},
})

var rootMutation = graphql.NewObject(graphql.ObjectConfig{
	Name: "RootMutation",
	Fields: graphql.Fields{
		"fee": &graphql.Field{
			Type: graphql.String,
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				return "baz", nil
			},
		},
	},
})

var schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	Query:    rootQuery,
	Mutation: rootMutation,
})

func executeQuery(query string, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})
	if len(result.Errors) > 0 {
		fmt.Printf("wrong result, unexpected errors: %v", result.Errors)
	}
	return result
}

func main() {
	e := casbin.NewEnforcer("./model.conf")
	e.EnableLog(true)

	// Having public access for feching schema
	e.AddPolicy("public", "__schema", ActionQuery)

	// Alice just has access to foo query
	e.AddPolicy("alice", "foo", ActionQuery)
	// Bob just has access to fee mutation
	e.AddPolicy("bob", "fee", Actionutation)

	// Alice and Bob have access to public resources
	e.AddGroupingPolicy("alice", "public")
	e.AddGroupingPolicy("bob", "public")

	cm := chainMiddleware(WithAuthorization(e), withTracing)

	http.HandleFunc("/graphql", cm(func(w http.ResponseWriter, r *http.Request) {
		var rb = struct {
			Query string `json:"query"`
		}{}
		json.NewDecoder(r.Body).Decode(&rb)
		result := executeQuery(rb.Query, schema)
		json.NewEncoder(w).Encode(result)
	}))
	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	http.ListenAndServe(":8080", nil)
}
