package agent

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/zeshi09/go_web_parser_agent/ent"
	"github.com/zeshi09/go_web_parser_agent/internal/storage"
)

func ScanAndNotifyDomains(ctx context.Context, client *ent.Client, c *storage.Cursor, webhook string, notify bool) error {
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
			if err := NotifyMMDomains(webhook, batch); err != nil {
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
func ScanAndNotifyLinks(ctx context.Context, client *ent.Client, c *storage.Cursor, webhook string, notify bool) error {
	total := 0
	for {
		batch, err := storage.CheckNewSocialLinks(ctx, client, *c)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		if notify {
			if err := NotifyMMLinks(webhook, batch); err != nil {
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
		log.Info().Int("new_links", total).Msg("processed")
	}
	return nil
}
