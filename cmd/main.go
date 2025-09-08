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

// пока храним курсоры для отслеживания изменений в базе данных в виде .json файлов
const (
	DomainStateFile = "domain_cursor.json"
	LinkStateFile   = "link_cursor.json"
)

// structure для курсора
var Curs storage.Cursor

// функции загрузки и сохранения данных в курсор .json
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

// основной луп для работы с доменами в таблице
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

// основной луп для работы с линками в таблице
func RunLoopLink(ctx context.Context, client *ent.Client, webhook string, interval time.Duration, sof bool) error {
	// загружаем курсор, чтобы просмотреть состояние изменений
	cur, err := loadCursor(LinkStateFile)
	if err != nil {
		return fmt.Errorf("failed to load link cursor: %w", err)
	}

	// сканируем таблицу и вызываем notify, если что-то изменилось
	if err := agent.ScanAndNotifyLinks(ctx, client, &cur, webhook, sof); err != nil {
		return fmt.Errorf("initial link scan failed: %w", err)
	}
	if err := saveCursor(cur, LinkStateFile); err != nil {
		return fmt.Errorf("failed to save link cursor: %w", err)
	}

	log.Debug().Msg("мы в initial scan")

	// заводим тикер для лупа
	t := time.NewTicker(interval)
	defer t.Stop()

	// основной цикл
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
			log.Debug().Msg("мы в ticker scan")
		}
	}
}

func main() {
	// обозначаем время в формате unix для логов
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// подгружаем .env файл, в котором хранятся все переменные для базы и мм
	err := godotenv.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load .env")
	}

	// подгружаем значение webhook url для мм из .env
	webhook := os.Getenv("MM_WEBHOOK")
	if webhook == "" {
		log.Fatal().Msg("MM_WEBHOOK env var is required")
	}

	// открываем базовый клиент для postgres с вызовом DSN метода для определения строки подключения из db.go
	client, err := ent.Open("postgres", storage.LoadConfigFromEnv().DSN())
	if err != nil {
		log.Error().Err(err).Msg("Failed to create db client")
	}
	defer client.Close()

	// обозначаем context
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// обозначаем переменные для запросов
	interval := 30 * time.Second
	sendOnFirst := false

	// канал с ошибками для корректной обработки горутин
	errCh := make(chan error, 2)

	// открываем две горутины, которые параллельно будут проверять таблицу с доменами и ссылками
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

	// обрабатываем завершение работы горутин (вообще завершение лупа не предполагается, только если произошла ошибка)
	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down...")
	case err := <-errCh:
		log.Error().Err(err).Msg("app err")
		cancel()
	}
}
