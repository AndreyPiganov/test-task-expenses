package agent

type toolDefinition struct {
	Type     string             `json:"type"`
	Function functionDefinition `json:"function"`
}

type functionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Strict      bool           `json:"strict"`
	Parameters  map[string]any `json:"parameters"`
}

func expenseTools() []toolDefinition {
	return []toolDefinition{
		newTool(
			toolCreateExpense,
			"Создать новый расход. Используй только когда известна сумма; дату и валюту подставляй по правилам полей.",
			map[string]any{
				"amount": map[string]any{
					"type":        "string",
					"description": "Положительная десятичная сумма в исходной валюте без пересчёта.",
				},
				"currency": map[string]any{
					"type":        "string",
					"description": "Трёхбуквенный код ISO 4217 в верхнем регистре. Если валюта не указана, RUB.",
				},
				"category": categorySchema(false, "Категория расхода."),
				"spent_at": map[string]any{
					"type":        "string",
					"description": "Дата расхода в формате YYYY-MM-DD. Если дата не указана, текущая дата.",
				},
				"comment": map[string]any{
					"type":        "string",
					"description": "Краткое назначение расхода без суммы, валюты и даты. Пустая строка, если назначения нет.",
				},
			},
			"amount", "currency", "category", "spent_at", "comment",
		),
		newTool(
			toolListExpenses,
			"Показать расходы пользователя по необязательным фильтрам.",
			filterProperties(),
			"category", "date_from", "date_to", "comment_query",
		),
		newTool(
			toolSummarizeExpenses,
			"Посчитать общую сумму и количество расходов пользователя по необязательным фильтрам.",
			filterProperties(),
			"category", "date_from", "date_to", "comment_query",
		),
		newTool(
			toolUpdateExpense,
			"Изменить один существующий расход. Не придумывай фильтры или новые значения.",
			map[string]any{
				"selector":      selectorSchema(),
				"category":      categorySchema(true, "Текущая категория для поиска расхода или null."),
				"date_from":     optionalStringSchema("Начало периода поиска YYYY-MM-DD или null."),
				"date_to":       optionalStringSchema("Конец периода поиска YYYY-MM-DD или null."),
				"comment_query": optionalStringSchema("Текст для поиска в комментарии или null."),
				"new_amount":    optionalStringSchema("Новая сумма в рублях или null, если сумма не меняется."),
				"new_category":  categorySchema(true, "Новая категория или null, если категория не меняется."),
				"new_spent_at":  optionalStringSchema("Новая дата YYYY-MM-DD или null, если дата не меняется."),
				"new_comment":   optionalStringSchema("Новый комментарий или null, если комментарий не меняется."),
				"clear_comment": map[string]any{
					"type":        "boolean",
					"description": "true, только если пользователь явно просит удалить комментарий.",
				},
			},
			"selector", "category", "date_from", "date_to", "comment_query",
			"new_amount", "new_category", "new_spent_at", "new_comment", "clear_comment",
		),
		newTool(
			toolDeleteExpense,
			"Удалить один существующий расход. Используй только при явной просьбе удалить расход.",
			map[string]any{
				"selector":      selectorSchema(),
				"category":      categorySchema(true, "Категория для поиска расхода или null."),
				"date_from":     optionalStringSchema("Начало периода поиска YYYY-MM-DD или null."),
				"date_to":       optionalStringSchema("Конец периода поиска YYYY-MM-DD или null."),
				"comment_query": optionalStringSchema("Текст для поиска в комментарии или null."),
			},
			"selector", "category", "date_from", "date_to", "comment_query",
		),
		newTool(
			toolRequestClarification,
			"Задать короткий уточняющий вопрос, если без ответа нельзя безопасно выбрать операцию или её аргументы.",
			map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "Уточняющий вопрос пользователю на русском языке.",
				},
			},
			"message",
		),
		newTool(
			toolRespondUnsupported,
			"Ответить, что запрос не относится к учёту расходов и не поддерживается.",
			map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "Короткий ответ пользователю на русском языке.",
				},
			},
			"message",
		),
	}
}

func newTool(
	name string,
	description string,
	properties map[string]any,
	required ...string,
) toolDefinition {
	return toolDefinition{
		Type: "function",
		Function: functionDefinition{
			Name:        name,
			Description: description,
			Strict:      true,
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           properties,
				"required":             required,
				"additionalProperties": false,
			},
		},
	}
}

func filterProperties() map[string]any {
	return map[string]any{
		"category":      categorySchema(true, "Категория для фильтрации или null."),
		"date_from":     optionalStringSchema("Начало периода YYYY-MM-DD или null."),
		"date_to":       optionalStringSchema("Конец периода YYYY-MM-DD или null."),
		"comment_query": optionalStringSchema("Текст для поиска в комментарии или null."),
	}
}

func optionalStringSchema(description string) map[string]any {
	return map[string]any{
		"type":        []string{"string", "null"},
		"description": description,
	}
}

func categorySchema(nullable bool, description string) map[string]any {
	values := []any{"еда", "транспорт", "жильё", "развлечения", "здоровье", "покупки", "другое"}
	typeValue := any("string")
	if nullable {
		typeValue = []string{"string", "null"}
		values = append(values, nil)
	}

	return map[string]any{
		"type":        typeValue,
		"enum":        values,
		"description": description,
	}
}

func selectorSchema() map[string]any {
	return map[string]any{
		"type": "string",
		"enum": []string{selectorLast, selectorSingle},
		"description": "last — только если пользователь явно сказал «последний» или «предыдущий»; " +
			"иначе single и точные фильтры.",
	}
}
