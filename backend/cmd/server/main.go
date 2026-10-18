package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/agent"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/config"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/currency"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/database"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/httpapi"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/repository/postgres"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/telegram"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger); err != nil {
		logger.Error("server stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("close database connection", "error", closeErr)
		}
	}()

	if err := database.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	logger.Info("PostgreSQL connection established and migrations applied")

	memberRepository := postgres.NewFamilyMemberRepository(db)
	expenseRepository := postgres.NewExpenseRepository(db)
	memberService := service.NewFamilyMemberService(memberRepository)
	expenseService := service.NewExpenseService(expenseRepository, memberRepository)

	var messageProcessor telegram.MessageProcessor
	if cfg.LLMBaseURL == "" || cfg.LLMModel == "" {
		logger.Info("LLM expense processor is disabled: LLM_BASE_URL or LLM_MODEL is empty")
	} else {
		llmClient := agent.NewClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
		messageProcessor = agent.NewProcessor(
			llmClient,
			expenseService,
			cfg.AppTimezone,
			currency.NewCBRClient(),
		)
		logger.Info("LLM expense processor configured", "model", cfg.LLMModel)
	}

	botCtx, stopBot := context.WithCancel(ctx)
	defer stopBot()
	if cfg.TelegramBotToken == "" {
		logger.Info("Telegram bot is disabled: TELEGRAM_BOT_TOKEN is empty")
	} else {
		bot := telegram.NewBot(
			telegram.NewClient(cfg.TelegramBotToken),
			memberService,
			messageProcessor,
			logger,
		)
		go func() {
			if err := bot.Run(botCtx); err != nil {
				logger.Error("Telegram bot stopped with an error", "error", err)
			}
		}()
		logger.Info("Telegram bot started in long polling mode")
	}

	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewHandler(
			logger,
			func(ctx context.Context) error { return db.PingContext(ctx) },
			expenseService,
			memberService,
		),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	logger.Info(
		"HTTP server started",
		"address", listener.Addr().String(),
		"timezone", cfg.AppTimezone.String(),
	)

	select {
	case err = <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("shutting down HTTP server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	err = <-serverErr
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logger.Info("HTTP server stopped")
	return nil
}
