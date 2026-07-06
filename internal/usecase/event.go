package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/gabrielgcosta/ticketblast-core/pkg/logger"
	"go.uber.org/zap"
)

type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	DecrBy(ctx context.Context, key string, decrement int64) (int64, error)
	IncrBy(ctx context.Context, key string, increment int64) (int64, error)
}

type ListActiveEventsUseCase struct {
	repo  EventRepository
	cache CacheService
}

func NewListActiveEventsUseCase(repo EventRepository, cache CacheService) *ListActiveEventsUseCase {
	return &ListActiveEventsUseCase{
		repo:  repo,
		cache: cache,
	}
}

type ListActiveEventsOutput struct {
	Events []*entity.Event `json:"events"`
}

const activeEventsCacheKey = "events:active"
const activeEventsCacheTTL = 5 * time.Minute

func (uc *ListActiveEventsUseCase) Execute(ctx context.Context) (*ListActiveEventsOutput, error) {
	// 1. Try to fetch from cache (Redis)
	cachedData, err := uc.cache.Get(ctx, activeEventsCacheKey)
	if err == nil && cachedData != "" {
		var events []*entity.Event
		if err := json.Unmarshal([]byte(cachedData), &events); err == nil {
			logger.Log.Info("Cache Hit: retrieved active events from Redis", zap.String("key", activeEventsCacheKey))
			return &ListActiveEventsOutput{Events: events}, nil
		}
		logger.Log.Warn("Failed to unmarshal cached active events, falling back to database", zap.Error(err))
	}

	logger.Log.Info("Cache Miss: fetching active events from database", zap.String("key", activeEventsCacheKey))

	// 2. Fetch from Postgres repository (active means future events)
	events, err := uc.repo.ListActive(ctx, time.Now())
	if err != nil {
		return nil, err
	}

	// 3. Serialize and save back to cache with TTL
	serialized, err := json.Marshal(events)
	if err == nil {
		if err := uc.cache.Set(ctx, activeEventsCacheKey, serialized, activeEventsCacheTTL); err != nil {
			logger.Log.Warn("Failed to write active events to Redis cache", zap.Error(err))
		} else {
			logger.Log.Info("Successfully cached active events to Redis", zap.String("key", activeEventsCacheKey))
		}
	} else {
		logger.Log.Warn("Failed to marshal active events for caching", zap.Error(err))
	}

	return &ListActiveEventsOutput{Events: events}, nil
}
