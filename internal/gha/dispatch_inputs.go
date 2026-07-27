package gha

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// DispatchInput describes one workflow_dispatch input from the workflow YAML.
type DispatchInput struct {
	ID          string   `json:"id"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type,omitempty"` // string, choice, boolean, environment, number
	Required    bool     `json:"required"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// ParseWorkflowDispatchInputs extracts workflow_dispatch inputs from workflow YAML.
// canDispatch is true when the workflow declares a workflow_dispatch trigger.
func ParseWorkflowDispatchInputs(yamlContent string) (canDispatch bool, inputs []DispatchInput, err error) {
	yamlContent = strings.TrimSpace(yamlContent)
	if yamlContent == "" {
		return false, nil, nil
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &doc); err != nil {
		return false, nil, fmt.Errorf("parse workflow yaml: %w", err)
	}
	on, ok := doc["on"]
	if !ok {
		on, ok = doc["true"] // YAML 1.1 quirk: "on" may parse as boolean true
	}
	if !ok {
		return false, nil, nil
	}

	switch v := on.(type) {
	case string:
		return strings.TrimSpace(v) == "workflow_dispatch", nil, nil
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) == "workflow_dispatch" {
				return true, nil, nil
			}
			if m, ok := item.(map[string]any); ok {
				if raw, ok := m["workflow_dispatch"]; ok {
					return true, parseDispatchInputsMap(raw), nil
				}
			}
		}
		return false, nil, nil
	case map[string]any:
		raw, ok := v["workflow_dispatch"]
		if !ok {
			return false, nil, nil
		}
		return true, parseDispatchInputsMap(raw), nil
	default:
		return false, nil, nil
	}
}

func parseDispatchInputsMap(raw any) []DispatchInput {
	if raw == nil {
		return nil
	}
	// workflow_dispatch: null / empty
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	inputsRaw, ok := m["inputs"]
	if !ok || inputsRaw == nil {
		return nil
	}
	inputsMap, ok := inputsRaw.(map[string]any)
	if !ok {
		return nil
	}
	out := make([]DispatchInput, 0, len(inputsMap))
	for id, spec := range inputsMap {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		inp := DispatchInput{ID: id, Type: "string"}
		switch s := spec.(type) {
		case string:
			inp.Description = s
		case map[string]any:
			inp.Description = asString(s["description"])
			inp.Type = strings.TrimSpace(asString(s["type"]))
			if inp.Type == "" {
				inp.Type = "string"
			}
			inp.Required = asBool(s["required"])
			inp.Default = asString(s["default"])
			inp.Options = asStringSlice(s["options"])
		default:
			continue
		}
		out = append(out, inp)
	}
	// Stable-ish order: required first, then by id.
	sortDispatchInputs(out)
	return out
}

func sortDispatchInputs(inputs []DispatchInput) {
	for i := 0; i < len(inputs); i++ {
		for j := i + 1; j < len(inputs); j++ {
			swap := false
			if inputs[i].Required != inputs[j].Required {
				swap = !inputs[i].Required && inputs[j].Required
			} else if inputs[i].ID > inputs[j].ID {
				swap = true
			}
			if swap {
				inputs[i], inputs[j] = inputs[j], inputs[i]
			}
		}
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	default:
		return ""
	}
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	default:
		return false
	}
}

func asStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s := strings.TrimSpace(asString(item))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// MergeDispatchDefaults fills missing field values from input defaults / first option.
func MergeDispatchDefaults(inputs []DispatchInput, fields map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range fields {
		out[k] = v
	}
	for _, inp := range inputs {
		if _, ok := out[inp.ID]; ok {
			continue
		}
		if inp.Default != "" {
			out[inp.ID] = inp.Default
			continue
		}
		if inp.Type == "choice" && len(inp.Options) > 0 {
			out[inp.ID] = inp.Options[0]
		}
		if inp.Type == "boolean" {
			out[inp.ID] = "false"
		}
	}
	return out
}

// MissingRequiredDispatchInputs lists required input ids without a non-empty value.
func MissingRequiredDispatchInputs(inputs []DispatchInput, fields map[string]string) []string {
	var missing []string
	for _, inp := range inputs {
		if !inp.Required {
			continue
		}
		v := strings.TrimSpace(fields[inp.ID])
		if v == "" {
			missing = append(missing, inp.ID)
		}
	}
	return missing
}
