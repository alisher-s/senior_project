package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/nu/student-event-ticketing-platform/analytics/model"
	"github.com/nu/student-event-ticketing-platform/analytics/repository"
)

type Service struct {
	repo repository.StatsRepository
	rdb  *redis.Client
}

func New(repo repository.StatsRepository, rdb *redis.Client) *Service {
	return &Service{repo: repo, rdb: rdb}
}

func cacheKey(callerID uuid.UUID, isAdmin bool, eventID *uuid.UUID) string {
	// Every key must reflect who is asking (and admin vs organizer for single-event stats).
	// Otherwise Redis could return another organizer’s cached body before the repository
	// runs its ownership check. Admin aggregate is one shared key: all admins see the same totals.
	scope := "organizer"
	if isAdmin {
		scope = "admin"
	}
	if eventID != nil {
		return fmt.Sprintf("analytics:event_stats:event:%s:caller:%s:%s", eventID.String(), callerID.String(), scope)
	}
	if isAdmin {
		return "analytics:event_stats:aggregate:admin"
	}
	return fmt.Sprintf("analytics:event_stats:aggregate:organizer:%s", callerID.String())
}

func (s *Service) EventStats(ctx context.Context, callerID uuid.UUID, isAdmin bool, eventID *uuid.UUID) (model.EventStats, error) {
	key := cacheKey(callerID, isAdmin, eventID)
	if s.rdb != nil {
		raw, err := s.rdb.Get(ctx, key).Bytes()
		if err == nil {
			var cached model.EventStats
			if err := json.Unmarshal(raw, &cached); err != nil {
				slog.Warn("analytics cache: invalid JSON, evicting key", "key", key, "err", err)
				_ = s.rdb.Del(ctx, key).Err()
			} else {
				return cached, nil
			}
		} else if err != nil && !errors.Is(err, redis.Nil) {
			// Fallback to DB if Redis is unavailable.
		}
	}

	stats, err := s.repo.EventStats(ctx, repository.EventStatsParams{
		CallerID: callerID,
		IsAdmin:  isAdmin,
		EventID:  eventID,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.EventStats{}, ErrEventNotFound
		}
		if errors.Is(err, repository.ErrForbidden) {
			return model.EventStats{}, ErrForbidden
		}
		return model.EventStats{}, err
	}

	if s.rdb != nil {
		b, err := json.Marshal(stats)
		if err == nil {
			_ = s.rdb.Set(ctx, key, b, 30*time.Second).Err()
		}
	}
	return stats, nil
}
