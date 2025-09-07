package storage

import (
	"context"
	"fmt"

	// "net/url"
	"os"
	// "strings"
	"time"

	// "entgo.io/ent/dialect"
	// "entgo.io/ent/dialect/sql"

	_ "github.com/lib/pq"

	"github.com/zeshi09/go_web_parser_agent/ent"
	"github.com/zeshi09/go_web_parser_agent/ent/domain"
	"github.com/zeshi09/go_web_parser_agent/ent/sociallink"
	// "github.com/zeshi09/go_web_parser_agent/ent/sociallink"
)

const PageSize = 500

type Cursor struct {
	LastCreatedAt time.Time `json:"last_created_at"`
	LastID        int       `json:"last_id"`
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type SocialLinkService struct {
	client *ent.Client
}

type DomainService struct {
	client *ent.Client
}

func LoadConfigFromEnv() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}
}

func (cfg *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)
}

func CheckDomains(ctx context.Context, client *ent.Client) ([]*ent.Domain, error) {
	d, err := client.Domain.
		Query().
		Select(domain.FieldLandingDomain).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed querying user: %w", err)
	}
	return d, nil
}

func CheckNewSocialLinks(ctx context.Context, client *ent.Client, cur Cursor) ([]*ent.SocialLink, error) {
	q := client.SocialLink.
		Query().
		Where(
			sociallink.Or(
				sociallink.CreatedAt(cur.LastCreatedAt),
				sociallink.And(
					sociallink.CreatedAtGT(cur.LastCreatedAt),
					sociallink.IDGT(cur.LastID),
				),
			),
		).
		Order(ent.Asc(sociallink.FieldCreatedAt),
			ent.Asc(sociallink.FieldID),
		).
		Limit(PageSize)
	return q.All(ctx)
}

func CheckNewDomains(ctx context.Context, client *ent.Client, cur Cursor) ([]*ent.Domain, error) {
	q := client.Domain.
		Query().
		Where(
			domain.Or(
				domain.CreatedAtGT(cur.LastCreatedAt),
				domain.And(
					domain.CreatedAtEQ(cur.LastCreatedAt),
					domain.IDGT(cur.LastID),
				),
			),
		).
		Order(
			ent.Asc(domain.FieldCreatedAt),
			ent.Asc(domain.FieldID),
		).
		Limit(PageSize)
	return q.All(ctx)
}
