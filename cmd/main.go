package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zeshi09/go_web_parser_agent/ent"

	"github.com/joho/godotenv"
	"github.com/zeshi09/go_web_parser_agent/internal/agent"
	"github.com/zeshi09/go_web_parser_agent/internal/storage"
)

const (
	DomainStateFile = "domain_cursor.json"
	LinkStateFile   = "link_cursor.json"
)

// type ForMM struct {
// 	Id     int    `json:"id"`
// 	Domain string `json:"domain"`
// }

var Curs storage.Cursor

func loadCursor(filename string) (storage.Cursor, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.Cursor{
				LastCreatedAt: time.Time{},
				LastID:        0,
			}, nil
		}
		return storage.Cursor{}, err
	}

	var cursor storage.Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return storage.Cursor{}, err
	}

	return cursor, nil
}

func saveCursor(cursor storage.Cursor, filename string) error {
	data, err := json.Marshal(cursor)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func RunLoopDomain(ctx context.Context, client *ent.Client, webhook string, interval time.Duration, sof bool) error {
	cur, err := loadCursor(DomainStateFile)
	if err != nil {
		return fmt.Errorf("failed to load domain cursor: %w", err)
	}

	if err := agent.ScanAndNotifyDomains(ctx, client, &cur, webhook, sof); err != nil {
		return fmt.Errorf("initial domain scan failed: %w", err)
	}
	if err := saveCursor(cur, DomainStateFile); err != nil {
		return fmt.Errorf("failed to save domain cursor: %w", err)
	}

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := agent.ScanAndNotifyDomains(ctx, client, &cur, webhook, true); err != nil {
				log.Error().Err(err).Msg("periodic scan failed")
				continue
			}
			if err := saveCursor(cur, DomainStateFile); err != nil {
				log.Error().Err(err).Msg("save cursor failed")
			}
		}
	}
}

func RunLoopLink(ctx context.Context, client *ent.Client, webhook string, interval time.Duration, sof bool) error {
	cur, err := loadCursor(LinkStateFile)
	if err != nil {
		return fmt.Errorf("failed to load link cursor: %w", err)
	}

	if err := agent.ScanAndNotifyLinks(ctx, client, &cur, webhook, sof); err != nil {
		return fmt.Errorf("initial link scan failed: %w", err)
	}
	if err := saveCursor(cur, LinkStateFile); err != nil {
		return fmt.Errorf("failed to save link cursor: %w", err)
	}

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := agent.ScanAndNotifyLinks(ctx, client, &cur, webhook, true); err != nil {
				log.Error().Err(err).Msg("periodic scan failed")
				continue
			}
			if err := saveCursor(cur, LinkStateFile); err != nil {
				log.Error().Err(err).Msg("save cursor failed")
			}
		}
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	err := godotenv.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load .env")
	}

	webhook := os.Getenv("MM_WEBHOOK")
	if webhook == "" {
		log.Fatal().Msg("MM_WEBHOOK env var is required")
	}

	client, err := ent.Open("postgres", storage.LoadConfigFromEnv().DSN())

	if err != nil {
		log.Error().Err(err).Msg("Failed to create db client")
	}
	defer client.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	interval := 30 * time.Second
	sendOnFirst := false

	errCh := make(chan error, 2)

	go func() {
		if err := RunLoopDomain(ctx, client, webhook, interval, sendOnFirst); err != nil {
			log.Error().Err(err).Msg("loop failed")
			errCh <- err
		}
	}()
	go func() {
		if err := RunLoopLink(ctx, client, webhook, interval, sendOnFirst); err != nil {
			log.Error().Err(err).Msg("loop failed")
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down...")
	case err := <-errCh:
		log.Error().Err(err).Msg("app err")
		cancel()
	}

}
