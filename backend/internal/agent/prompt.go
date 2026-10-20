package agent

import (
	_ "embed"
	"strings"
	"time"
)

//go:embed prompts/system.txt
var systemPromptTemplate string

func systemPrompt(now time.Time) string {
	return strings.NewReplacer(
		"{{CURRENT_DATE}}", now.Format(dateLayout),
		"{{TIMEZONE}}", now.Location().String(),
	).Replace(systemPromptTemplate)
}
