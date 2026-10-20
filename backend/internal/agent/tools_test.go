package agent

import "testing"

func TestExpenseToolsUseStrictObjectSchemas(t *testing.T) {
	t.Parallel()

	tools := expenseTools()
	if len(tools) != 7 {
		t.Fatalf("tools = %d, want 7", len(tools))
	}

	seen := make(map[string]bool, len(tools))
	for _, tool := range tools {
		if tool.Type != "function" {
			t.Errorf("tool %q type = %q, want function", tool.Function.Name, tool.Type)
		}
		if tool.Function.Name == "" || seen[tool.Function.Name] {
			t.Errorf("tool name %q is empty or duplicated", tool.Function.Name)
		}
		seen[tool.Function.Name] = true
		if !tool.Function.Strict {
			t.Errorf("tool %q is not strict", tool.Function.Name)
		}
		if additional, ok := tool.Function.Parameters["additionalProperties"].(bool); !ok || additional {
			t.Errorf("tool %q allows additional properties", tool.Function.Name)
		}

		properties, ok := tool.Function.Parameters["properties"].(map[string]any)
		if !ok {
			t.Errorf("tool %q has no object properties", tool.Function.Name)
			continue
		}
		required, ok := tool.Function.Parameters["required"].([]string)
		if !ok || len(required) != len(properties) {
			t.Errorf(
				"tool %q required fields = %d, properties = %d",
				tool.Function.Name,
				len(required),
				len(properties),
			)
		}
	}
}
