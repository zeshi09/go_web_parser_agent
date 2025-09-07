package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zeshi09/go_web_parser_agent/ent"

	// "github.com/zeshi09/go_web_parser_agent/ent/domain"
	"github.com/joho/godotenv"
	"github.com/zeshi09/go_web_parser_agent/internal/agent"
	"github.com/zeshi09/go_web_parser_agent/internal/storage"
)

type ForMM struct {
	Id     int    `json:"id"`
	Domain string `json:"domain"`
}

var Curs storage.Cursor

func loadCursor(c *storage.Cursor) (storage.Cursor, error) {
	return *c, nil
}

func saveCursor(c *storage.Cursor) error {
	Curs = *c
	return nil
}

func RunLoop(ctx context.Context, client *ent.Client, stateFile storage.Cursor, webhook string, interval time.Duration, sof bool) error {
	cur, err := loadCursor(&Curs)
	if err != nil {
		return err
	}

	if err := agent.ScanAndNotifyDomains(ctx, client, &cur, webhook, sof); err != nil {
		return err
	}
	if err := saveCursor(&cur); err != nil {
		return err
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
			if err := saveCursor(&cur); err != nil {
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

	client, err := ent.Open("postgres", storage.LoadConfigFromEnv().DSN())
	if err != nil {
		log.Error().Err(err).Msg("Failed to create db client")
	}

	defer client.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	stateFile := Curs
	interval := 30 * time.Second
	sendOnFirst := false

	if err := RunLoop(ctx, client, stateFile, webhook, interval, sendOnFirst); err != nil {
		log.Error().Err(err).Msg("loop failed")
	}

}
