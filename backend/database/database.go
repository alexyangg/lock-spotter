package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"lockspotter-backend/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// StorageClient aggregates our active database connection pools
type StorageClient struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

// InitStorage Layer opens concurrent connections to both operational data layers
func InitStorage(cfg *config.Config) (*StorageClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Establish PostgreSQL Connection Pool via pgxpool
	log.Println("[*] Allocating PostgreSQL thread-safe connection pool...")
	pgConfig, err := pgxpool.ParseConfig(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database string: %w", err)
	}

	// Tweak connection limits for high-throughput scaling
	pgConfig.MaxConns = 25
	pgConfig.MinConns = 5

	dbPool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return nil, fmt.Errorf("postgres pool allocation failed: %w", err)
	}

	// Verify Postgres is reachable
	if err := dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping verification failed: %w", err)
	}
	log.Println("[+] PostgreSQL connection verified and pooled.")

	// 2. Establish Redis Client
	log.Println("[*] Connecting to Redis in-memory ring buffer...")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: "", // No password set in local docker compose
		DB:       0,  // Default logical storage block
	})

	// Verify Redis is reachable
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping verification failed: %w", err)
	}
	log.Println("[+] Redis operational status verified.")

	return &StorageClient{
		DB:    dbPool,
		Redis: redisClient,
	}, nil
}

// Close gracefully flushes remaining transactions and terminates connections
func (s *StorageClient) Close() {
	if s.DB != nil {
		s.DB.Close()
		log.Println("[-] PostgreSQL connection pool drained safely.")
	}
	if s.Redis != nil {
		s.Redis.Close()
		log.Println("[-] Redis client context severed safely.")
	}
}