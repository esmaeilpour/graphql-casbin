package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/casbin/casbin"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
)

type middleware func(next http.HandlerFunc) http.HandlerFunc

var (
	ActionQuery   = "Query"
	Actionutation = "Mutation"
)

func WithAuthorization(ef *casbin.Enforcer) middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Read the content
			var bodyBytes []byte
			if r != nil {
				bodyBytes, _ = ioutil.ReadAll(r.Body)
			}
			// Restore the io.ReadCloser to its original state
			r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

			var rb = struct {
				Query string `json:"query"`
			}{}
			json.Unmarshal(bodyBytes, &rb)

			// We have got subject from custom header
			// in real production subject could be in
			// something like jwt claim etc
			var subject = r.Header.Get("X-Subject")

			doc, _ := parser.Parse(parser.ParseParams{Source: rb.Query})
			for _, node := range doc.Definitions {
				switch d := node.(type) {
				case ast.TypeSystemDefinition:
					{
						var o string
						switch d.GetOperation() {
						case ast.OperationTypeQuery:
							o = ActionQuery
						case ast.OperationTypeMutation:
							o = Actionutation
						default:
							continue
						}
						for _, s := range d.GetSelectionSet().Selections {
							switch f := s.(type) {
							case *ast.Field:
								if !ef.Enforce(subject, f.Name.Value, o) {
									w.WriteHeader(http.StatusForbidden)
									w.Header().Add("Content-Type", "application/json")
									res, _ := json.Marshal(map[string]string{
										"error": "Forbidden",
									})
									w.Write(res)
									return
								}
							}
						}
					}
				}
			}
			next(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func withTracing(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// log something
		next.ServeHTTP(w, r)
	}
}

// chainMiddleware provides syntactic sugar to create a new middleware
// which will be the result of chaining the ones received as parameters.
func chainMiddleware(mw ...middleware) middleware {
	return func(final http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			last := final
			for i := len(mw) - 1; i >= 0; i-- {
				last = mw[i](last)
			}
			last(w, r)
		}
	}
}
