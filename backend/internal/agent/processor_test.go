package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/currency"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestProcessorCreatesExpenseForTelegramUser(t *testing.T) {
	t.Parallel()

	timezone, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	llm := &fakeLLM{response: `{
		"action":"create_expense",
		"amount":"1200",
		"category":"транспорт",
		"spent_at":"2026-10-15",
		"comment":"такси",
		"question":""
	}`}
	expenses := &fakeExpenseCreator{}
	processor := NewProcessor(llm, expenses, timezone)
	processor.now = func() time.Time {
		return time.Date(2026, time.October, 16, 12, 0, 0, 0, timezone)
	}

	response, err := processor.Process(context.Background(), 123, "такси 1200 вчера")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if expenses.input.UserID != 123 {
		t.Errorf("UserID = %d, want Telegram user 123", expenses.input.UserID)
	}
	if expenses.input.SpentAt.Format(dateLayout) != "2026-10-15" {
		t.Errorf("SpentAt = %s, want yesterday", expenses.input.SpentAt)
	}
	if expenses.input.Category != domain.CategoryTransport {
		t.Errorf("Category = %q, want transport", expenses.input.Category)
	}
	if !strings.Contains(response, "такси — 1200 ₽") {
		t.Errorf("response = %q, want created expense confirmation", response)
	}
	if !strings.Contains(llm.systemPrompt, "2026-10-16") {
		t.Errorf("system prompt does not contain current date: %q", llm.systemPrompt)
	}
	if !strings.Contains(llm.systemPrompt, "краткое назначение расхода") {
		t.Error("system prompt does not explain how to extract an expense comment")
	}
	if !strings.Contains(llm.systemPrompt, "самостоятельно ничего не пересчитывай") {
		t.Error("system prompt does not delegate currency conversion to the backend")
	}
	if !strings.Contains(llm.systemPrompt, "поле currency обязательно") ||
		!strings.Contains(llm.systemPrompt, "код ISO 4217") {
		t.Error("system prompt does not require an ISO 4217 currency code")
	}
	if !strings.Contains(llm.systemPrompt, `«я потратил 500» → amount="500", currency="RUB"`) {
		t.Error("system prompt does not default an unspecified currency to RUB")
	}
}

func TestProcessorConvertsForeignCurrencyExpenseToRubles(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{response: `{
		"action":"create_expense",
		"amount":"100",
		"currency":"USD",
		"category":"покупки",
		"spent_at":"2026-10-18",
		"comment":"наушники"
	}`}
	expenses := &fakeExpenseCreator{}
	converter := &fakeCurrencyConverter{conversion: currency.Conversion{
		AmountRUB: decimal.RequireFromString("8125.50"),
		Rate:      decimal.RequireFromString("81.255"),
		RateDate:  time.Date(2026, time.October, 17, 0, 0, 0, 0, time.UTC),
	}}
	processor := NewProcessor(llm, expenses, time.UTC, converter)

	response, err := processor.Process(context.Background(), 123, "купил наушники за 100 долларов")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if converter.currencyCode != "USD" || !converter.amount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("converter input = %s %s, want 100 USD", converter.amount, converter.currencyCode)
	}
	if got := converter.date.Format(dateLayout); got != "2026-10-18" {
		t.Errorf("converter date = %s, want 2026-10-18", got)
	}
	if !expenses.input.Amount.Equal(decimal.RequireFromString("8125.50")) {
		t.Errorf("saved amount = %s, want 8125.50 RUB", expenses.input.Amount)
	}
	if expenses.input.Comment != "наушники" {
		t.Errorf("saved comment = %q, want наушники", expenses.input.Comment)
	}
	if !strings.Contains(response, "8125.5 ₽") || !strings.Contains(response, "из 100 USD") ||
		!strings.Contains(response, "17.10.2026") {
		t.Errorf("response = %q, want conversion details", response)
	}
}

func TestProcessorRecognizesHryvniaFromMessageWhenLLMMissesCurrency(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{response: `{
		"action":"create_expense",
		"amount":"500",
		"currency":"RUB",
		"category":"еда",
		"spent_at":"2026-10-19",
		"comment":"продукты"
	}`}
	expenses := &fakeExpenseCreator{}
	converter := &fakeCurrencyConverter{conversion: currency.Conversion{
		AmountRUB: decimal.RequireFromString("931.94"),
		Rate:      decimal.RequireFromString("1.86387"),
		RateDate:  time.Date(2026, time.October, 17, 0, 0, 0, 0, time.UTC),
	}}
	processor := NewProcessor(llm, expenses, time.UTC, converter)

	response, err := processor.Process(context.Background(), 123, "Купил продуктов на 500 гривен")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if converter.currencyCode != "UAH" {
		t.Fatalf("converter currency = %q, want UAH", converter.currencyCode)
	}
	if !expenses.input.Amount.Equal(decimal.RequireFromString("931.94")) {
		t.Errorf("saved amount = %s, want 931.94 RUB", expenses.input.Amount)
	}
	if !strings.Contains(response, "из 500 UAH") {
		t.Errorf("response = %q, want UAH conversion details", response)
	}
}

func TestProcessorRecognizesTengeFromMessageWhenLLMMissesCurrency(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{response: `{
		"action":"create_expense",
		"amount":"1000",
		"currency":"RUB",
		"category":"покупки",
		"spent_at":"2026-10-19",
		"comment":"сувениры"
	}`}
	expenses := &fakeExpenseCreator{}
	converter := &fakeCurrencyConverter{conversion: currency.Conversion{
		AmountRUB: decimal.RequireFromString("184.12"),
		Rate:      decimal.RequireFromString("0.184123"),
		RateDate:  time.Date(2026, time.October, 17, 0, 0, 0, 0, time.UTC),
	}}
	processor := NewProcessor(llm, expenses, time.UTC, converter)

	response, err := processor.Process(context.Background(), 123, "Купил сувениры за 1000 тенге")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if converter.currencyCode != "KZT" {
		t.Fatalf("converter currency = %q, want KZT", converter.currencyCode)
	}
	if !expenses.input.Amount.Equal(decimal.RequireFromString("184.12")) {
		t.Errorf("saved amount = %s, want 184.12 RUB", expenses.input.Amount)
	}
	if !strings.Contains(response, "из 1000 KZT") {
		t.Errorf("response = %q, want KZT conversion details", response)
	}
}

func TestProcessorReturnsClarificationWithoutCreatingExpense(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{response: `{
		"action":"clarify",
		"amount":"",
		"category":"",
		"spent_at":"",
		"comment":"",
		"question":"Какую сумму вы потратили на кофе?"
	}`}
	expenses := &fakeExpenseCreator{}
	processor := NewProcessor(llm, expenses, time.UTC)

	response, err := processor.Process(context.Background(), 123, "кофе")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if response != "Какую сумму вы потратили на кофе?" {
		t.Errorf("response = %q, want clarification", response)
	}
	if expenses.called {
		t.Fatal("expense service was called for a clarification")
	}
}

func TestParseCommandRejectsAdditionalJSON(t *testing.T) {
	t.Parallel()

	if _, err := parseCommand(`{"action":"clarify"} {"action":"create_expense"}`); err == nil {
		t.Fatal("parseCommand() accepted more than one JSON value")
	}
}

func TestProcessorSummarizesOnlyTelegramUserFilter(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{response: `{
		"action":"summarize_expenses",
		"category":"еда",
		"date_from":"2026-10-01",
		"date_to":"2026-10-31"
	}`}
	expenses := &fakeExpenseCreator{listItems: []domain.Expense{
		{Amount: decimal.RequireFromString("350.50")},
		{Amount: decimal.RequireFromString("149.50")},
	}}
	processor := NewProcessor(llm, expenses, time.UTC)

	response, err := processor.Process(context.Background(), 777, "сколько потратил на еду в этом месяце")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if response != "Итого: 500 ₽. Количество расходов: 2." {
		t.Errorf("response = %q, want total", response)
	}
	if expenses.lastFilter.UserID == nil || *expenses.lastFilter.UserID != 777 {
		t.Fatal("expense list was not restricted by Telegram user ID")
	}
	if expenses.lastFilter.Category == nil || *expenses.lastFilter.Category != domain.CategoryFood {
		t.Fatal("category filter was not passed to expense service")
	}
}

func TestProcessorDoesNotDeleteAmbiguousExpense(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{response: `{
		"action":"delete_expense",
		"comment_query":"такси",
		"selector":"single"
	}`}
	expenses := &fakeExpenseCreator{listItems: []domain.Expense{
		{ID: uuid.New(), UserID: 123, Comment: "такси", Category: domain.CategoryTransport},
		{ID: uuid.New(), UserID: 123, Comment: "такси домой", Category: domain.CategoryTransport},
	}}
	processor := NewProcessor(llm, expenses, time.UTC)

	response, err := processor.Process(context.Background(), 123, "удали такси")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if !strings.Contains(response, "Нашёл несколько") {
		t.Errorf("response = %q, want ambiguity clarification", response)
	}
	if expenses.deleted {
		t.Fatal("ambiguous expense was deleted")
	}
}

func TestProcessorUpdatesLastExpenseForTelegramUser(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	llm := &fakeLLM{response: `{
		"action":"update_expense",
		"selector":"last",
		"new_amount":"450"
	}`}
	expenses := &fakeExpenseCreator{listItems: []domain.Expense{{
		ID:       id,
		UserID:   123,
		Amount:   decimal.NewFromInt(350),
		Category: domain.CategoryFood,
		SpentAt:  time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC),
		Comment:  "кофе",
	}}}
	processor := NewProcessor(llm, expenses, time.UTC)

	_, err := processor.Process(context.Background(), 123, "последнее было не 350, а 450")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if expenses.updatedID != id || expenses.updatedUserID != 123 {
		t.Fatal("UpdateForUser() did not receive the selected expense and Telegram user")
	}
	if expenses.updateInput.Amount == nil || !expenses.updateInput.Amount.Equal(decimal.NewFromInt(450)) {
		t.Fatal("new amount was not passed to UpdateForUser()")
	}
}

func TestProcessorRequiresExplicitDeleteIntent(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{response: `{
		"action":"delete_expense",
		"comment_query":"такси",
		"selector":"single"
	}`}
	expenses := &fakeExpenseCreator{listItems: []domain.Expense{{
		ID:       uuid.New(),
		UserID:   123,
		Comment:  "такси",
		Category: domain.CategoryTransport,
	}}}
	processor := NewProcessor(llm, expenses, time.UTC)

	response, err := processor.Process(context.Background(), 123, "покажи такси")
	if err != nil {
		t.Fatalf("Process() returned an error: %v", err)
	}
	if !strings.Contains(response, "явно напишите «удали»") {
		t.Errorf("response = %q, want explicit delete clarification", response)
	}
	if expenses.deleted {
		t.Fatal("expense was deleted without explicit user intent")
	}
}

type fakeLLM struct {
	response     string
	systemPrompt string
}

func (l *fakeLLM) Complete(_ context.Context, systemPrompt, _ string) (string, error) {
	l.systemPrompt = systemPrompt
	return l.response, nil
}

type fakeCurrencyConverter struct {
	conversion   currency.Conversion
	amount       decimal.Decimal
	currencyCode string
	date         time.Time
}

func (c *fakeCurrencyConverter) ToRubles(
	_ context.Context,
	amount decimal.Decimal,
	currencyCode string,
	date time.Time,
) (currency.Conversion, error) {
	c.amount = amount
	c.currencyCode = currencyCode
	c.date = date
	return c.conversion, nil
}

type fakeExpenseCreator struct {
	input         service.CreateExpenseInput
	called        bool
	listItems     []domain.Expense
	lastFilter    service.ExpenseFilter
	updatedID     uuid.UUID
	updatedUserID int64
	updateInput   service.UpdateExpenseInput
	deleted       bool
}

func (c *fakeExpenseCreator) Create(
	_ context.Context,
	input service.CreateExpenseInput,
) (domain.Expense, error) {
	c.called = true
	c.input = input
	return domain.Expense{
		ID:       uuid.New(),
		UserID:   input.UserID,
		Amount:   input.Amount,
		Category: input.Category,
		SpentAt:  input.SpentAt,
		Comment:  input.Comment,
	}, nil
}

func (c *fakeExpenseCreator) List(
	_ context.Context,
	filter service.ExpenseFilter,
) (service.ExpensePage, error) {
	c.lastFilter = filter
	items := c.listItems
	if filter.Offset >= len(items) {
		items = nil
	} else {
		items = items[filter.Offset:]
		if len(items) > filter.Limit {
			items = items[:filter.Limit]
		}
	}
	return service.ExpensePage{
		Items:  items,
		Total:  len(c.listItems),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (c *fakeExpenseCreator) UpdateForUser(
	_ context.Context,
	userID int64,
	id uuid.UUID,
	input service.UpdateExpenseInput,
) (domain.Expense, error) {
	c.updatedID = id
	c.updatedUserID = userID
	c.updateInput = input
	expense := c.listItems[0]
	if input.Amount != nil {
		expense.Amount = *input.Amount
	}
	return expense, nil
}

func (c *fakeExpenseCreator) DeleteForUser(_ context.Context, _ int64, _ uuid.UUID) error {
	c.deleted = true
	return nil
}
