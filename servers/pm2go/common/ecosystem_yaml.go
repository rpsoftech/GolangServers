package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ecosystemJSON returns the JSON form of an ecosystem file. YAML is parsed
// into generic data and re-marshalled so the JSON path (AppConfig.UnmarshalJSON,
// defaults, env_<profile>) is the only decoder. An unknown extension tries JSON
// first, then YAML.
func ecosystemJSON(path string, b []byte) ([]byte, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return b, nil
	case ".yaml", ".yml":
		return yamlToJSON(path, b)
	}
	if json.Valid(b) {
		return b, nil
	}
	out, err := yamlToJSON(path, b)
	if err != nil {
		return nil, fmt.Errorf("%s: not valid JSON or YAML (use .json, .yaml or .yml): %w", path, err)
	}
	return out, nil
}

func yamlToJSON(path string, b []byte) ([]byte, error) {
	var v any
	if err := yaml.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	v, err := normalizeYAML(v)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if m, ok := v.(map[string]any); ok {
		stringifyEnv(m)
		if apps, ok := m["apps"].([]any); ok {
			for _, a := range apps {
				if am, ok := a.(map[string]any); ok {
					stringifyEnv(am)
				}
			}
		}
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return out, nil
}

// normalizeYAML converts map[any]any (non-string keys) into map[string]any.
func normalizeYAML(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			n, err := normalizeYAML(e)
			if err != nil {
				return nil, err
			}
			t[k] = n
		}
		return t, nil
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, e := range t {
			n, err := normalizeYAML(e)
			if err != nil {
				return nil, err
			}
			m[fmt.Sprint(k)] = n
		}
		return m, nil
	case []any:
		for i, e := range t {
			n, err := normalizeYAML(e)
			if err != nil {
				return nil, err
			}
			t[i] = n
		}
		return t, nil
	}
	return v, nil
}

// stringifyEnv turns scalar env values (PORT: 3000, DEBUG: true) into strings,
// since env maps are string to string.
func stringifyEnv(app map[string]any) {
	for k, v := range app {
		if k != "env" && !strings.HasPrefix(k, "env_") {
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		for ek, ev := range m {
			switch ev.(type) {
			case nil, string, map[string]any, []any:
			default:
				m[ek] = fmt.Sprint(ev)
			}
		}
	}
}

func jsonKeys(t reflect.Type) map[string]bool {
	keys := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		name := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			keys[name] = true
		}
	}
	return keys
}

// unknownKeyWarnings lists keys of the JSON form that no field knows, so a
// typo such as max_restart is reported instead of silently ignored.
func unknownKeyWarnings(b []byte) []string {
	var top map[string]json.RawMessage
	if json.NewDecoder(bytes.NewReader(b)).Decode(&top) != nil {
		return nil
	}
	var warns []string
	known := jsonKeys(reflect.TypeOf(Ecosystem{}))
	for _, k := range sortedKeys(top) {
		if !known[k] {
			warns = append(warns, fmt.Sprintf("unknown top-level key %q", k))
		}
	}
	var apps []map[string]json.RawMessage
	if json.Unmarshal(top["apps"], &apps) != nil {
		return warns
	}
	appKeys := jsonKeys(reflect.TypeOf(AppConfig{}))
	for i, app := range apps {
		name := fmt.Sprintf("#%d", i+1)
		var n string
		if json.Unmarshal(app["name"], &n) == nil && n != "" {
			name = n
		} else {
			var s string
			if json.Unmarshal(app["script"], &s) == nil && DefaultName(s) != "" {
				name = DefaultName(s)
			}
		}
		for _, k := range sortedKeys(app) {
			if !appKeys[k] && !(strings.HasPrefix(k, "env_") && len(k) > 4) {
				warns = append(warns, fmt.Sprintf("app %q: unknown key %q", name, k))
			}
		}
	}
	return warns
}

func sortedKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// IsEcosystemFile reports whether a start target is an ecosystem file by its
// extension (.json, .yaml, .yml).
func IsEcosystemFile(target string) bool {
	switch strings.ToLower(filepath.Ext(target)) {
	case ".json", ".yaml", ".yml":
		return true
	}
	return false
}
