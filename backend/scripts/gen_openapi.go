// Package main validates cmd/server/docs/openapi.json before build (run in Dockerfile).
// Run from backend dir: go run ./scripts/gen_openapi.go
package main

import (
	"encoding/json"
	"os"
)

const specPath = "cmd/server/docs/openapi.json"

func main() {
	b, err := os.ReadFile(specPath)
	if err != nil {
		panic("openapi.json missing: " + err.Error())
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		panic("invalid openapi.json: " + err.Error())
	}
	if v, _ := m["openapi"]; v == nil {
		panic("openapi.json must set openapi version (e.g. 3.0.0)")
	}
	if paths, _ := m["paths"].(map[string]interface{}); paths == nil {
		panic("openapi.json must have paths object")
	}
}
