// Package mcpmerge adds the codebase-memory server to a .mcp.json and keeps every other key.
package mcpmerge

import (
	"bytes"
	"encoding/json"
	"errors"
)

func Merge(src []byte, bin string) ([]byte, error) {
	var data map[string]any
	if err := json.Unmarshal(src, &data); err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errors.New("not a JSON object")
	}
	servers, ok := data["mcpServers"].(map[string]any)
	if !ok {
		servers = map[string]any{}
	}
	servers["codebase-memory"] = map[string]any{"command": bin, "args": []string{"--ui=false"}, "env": map[string]any{}}
	data["mcpServers"] = servers
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(data)
	return b.Bytes(), err
}
