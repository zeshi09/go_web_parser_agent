package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zeshi09/go_web_parser_agent/ent"

	// "github.com/zeshi09/go_web_parser_agent/ent/domain"
	"github.com/joho/godotenv"
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

func ScanAndNotify(ctx context.Context, client *ent.Client, c *storage.Cursor, webhook string, notify bool) error {
	total := 0
	for {
		batch, err := storage.CheckNewDomains(ctx, client, *c)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		if notify {
			if err := notifyMM(webhook, batch); err != nil {
				return err
			}
		}

		last := batch[len(batch)-1]
		c.LastCreatedAt = last.CreatedAt
		c.LastID = last.ID

		total += len(batch)
		if len(batch) < storage.PageSize {
			break
		}
	}
	if total > 0 {
		log.Info().Int("new_domains", total).Msg("processed")
	}
	return nil
}

func notifyMM(webhook string, domains []*ent.Domain) error {
	var b strings.Builder
	b.WriteString("**Появились новые домены:**\n")
	for _, d := range domains {
		b.WriteString("- ")
		b.WriteString(d.LandingDomain)
		b.WriteString("\n")
	}
	payload := map[string]string{
		"text":     b.String(),
		"username": "DomainWatcher",
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("mm webhook returned %s", resp.Status)
	}
	return nil
}

func runLoop(ctx context.Context, client *ent.Client, stateFile storage.Cursor, webhook string, interval time.Duration, sof bool) error {
	cur, err := loadCursor(&Curs)
	if err != nil {
		return err
	}

	if err := ScanAndNotify(ctx, client, &cur, webhook, sof); err != nil {
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
			if err := ScanAndNotify(ctx, client, &cur, webhook, true); err != nil {
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

	if err := runLoop(ctx, client, stateFile, webhook, interval, sendOnFirst); err != nil {
		log.Error().Err(err).Msg("loop failed")
	}

	// domains, err := storage.CheckDomains(context.Background(), client)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to check domains")
	// }

	// for_mm := make(map[int]string)
	// text := ""
	// for i := range domains {
	// 	for_mm[i] = domains[i].LandingDomain
	// 	text += domains[i].LandingDomain + "\n"
	// }

	// fmt.Println(len(domains))

	// // jsonBytes, _ := json.MarshalIndent(for_mm, "", "")
	// payload := map[string]string{
	// 	"text": "Список доменов:\n" + text,
	// }
	// jsonBytes, _ := json.Marshal(payload)

	// fmt.Println(string(jsonBytes))

	// req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewBuffer(jsonBytes))
	// req.Header.Set("Content-Type", "application/json")
	// httpClient := &http.Client{}
	// response, err := httpClient.Do(req)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to fetch mm")
	// }
	// fmt.Println(response)
	// defer response.Body.Close()

}
