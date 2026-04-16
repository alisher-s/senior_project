package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	"github.com/nu/student-event-ticketing-platform/internal/config"
	"github.com/nu/student-event-ticketing-platform/internal/infra/db"
	authRepo "github.com/nu/student-event-ticketing-platform/auth/repository"
	authService "github.com/nu/student-event-ticketing-platform/auth/service"
	ticketingRepo "github.com/nu/student-event-ticketing-platform/ticketing/repository"
	ticketingService "github.com/nu/student-event-ticketing-platform/ticketing/service"
	paymentsRepo "github.com/nu/student-event-ticketing-platform/payments/repository"
	paymentsService "github.com/nu/student-event-ticketing-platform/payments/service"
	paymentsModel "github.com/nu/student-event-ticketing-platform/payments/model"
	ticketingModel "github.com/nu/student-event-ticketing-platform/ticketing/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func connectDBOrSkip(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Skipf("config not available: %v", err)
	}

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		t.Skipf("postgres not reachable: %v", err)
	}

	// Verify expected schema exists.
	var one int
	if err := pool.QueryRow(ctx, `SELECT 1 FROM events LIMIT 1`).Scan(&one); err != nil {
		pool.Close()
		t.Skipf("expected schema not present: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return ctx, pool
}

// NOTE: integration tests require DB access and may be skipped.
func TestTicketCapacityRace(t *testing.T) {
	ctx, pool := connectDBOrSkip(t)
	var err error

	// Fresh state.
	_, _ = pool.Exec(ctx, `
		TRUNCATE TABLE payments, tickets, events, refresh_tokens, users RESTART IDENTITY CASCADE
	`)

	eventID := uuid.New()
	capacityTotal := 2
	_, err = pool.Exec(ctx, `
		INSERT INTO events (id, title, description, starts_at, capacity_total, capacity_available)
		VALUES ($1, 'e', 'd', NOW(), $2, $2)
	`, eventID, capacityTotal)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	users := make([]uuid.UUID, 5)
	for i := range users {
		users[i] = uuid.New()
		email := "u" + uuid.NewString() + "@nu.edu.kz"
		_, err := pool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, role)
			VALUES ($1, $2, 'x', 'student')
		`, users[i], email)
		if err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	repo := ticketingRepo.NewPostgres(pool)
	svc := ticketingService.New(repo)

	var wg sync.WaitGroup
	successCh := make(chan int, len(users))
	errCh := make(chan error, len(users))

	for _, userID := range users {
		wg.Add(1)
		go func(uid uuid.UUID) {
			defer wg.Done()
			_, _, err := svc.RegisterTicket(ctx, uid, eventID)
			if err == nil {
				successCh <- 1
				return
			}
			errCh <- err
		}(userID)
	}

	wg.Wait()
	close(successCh)
	close(errCh)

	successes := 0
	for range successCh {
		successes++
	}
	_ = errCh

	// Capacity should never go below 0 and should match successful seats.
	var capacityAvailable int
	if err := pool.QueryRow(ctx, `SELECT capacity_available FROM events WHERE id=$1`, eventID).Scan(&capacityAvailable); err != nil {
		t.Fatalf("query capacity_available: %v", err)
	}

	if capacityAvailable < 0 {
		t.Fatalf("capacity_available < 0: %d", capacityAvailable)
	}
	if capacityAvailable != capacityTotal-successes {
		t.Fatalf("capacity mismatch: capacity_available=%d successes=%d", capacityAvailable, successes)
	}

	var ticketsCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM tickets WHERE event_id=$1 AND status='active'`, eventID).Scan(&ticketsCount); err != nil {
		t.Fatalf("query tickets count: %v", err)
	}
	if ticketsCount != successes {
		t.Fatalf("tickets count mismatch: tickets=%d successes=%d", ticketsCount, successes)
	}
}

func TestAuthRefreshSingleUse(t *testing.T) {
	ctx, pool := connectDBOrSkip(t)
	cfg, _ := config.LoadFromEnv()

	_, _ = pool.Exec(ctx, `
		TRUNCATE TABLE payments, tickets, events, refresh_tokens, users RESTART IDENTITY CASCADE
	`)

	userRepo := authRepo.NewPostgres(pool)
	jwt := authx.NewJWT(cfg)
	svc := authService.New(cfg, userRepo, userRepo, jwt)

	email := "student_" + uuid.NewString() + "@nu.edu.kz"
	_, access, refreshToken, err := svc.Register(ctx, email, "verystrongpassword")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_ = access

	_, _, newRefresh, err := svc.Refresh(ctx, refreshToken)
	if err != nil {
		t.Fatalf("refresh #1: %v", err)
	}
	if newRefresh == refreshToken {
		t.Fatalf("expected refresh rotation, but refresh token did not change")
	}

	_, _, _, err = svc.Refresh(ctx, refreshToken)
	if err == nil {
		t.Fatalf("expected refresh token reuse to fail")
	}
	if !errors.Is(err, authService.ErrRefreshTokenConsumed) {
		t.Fatalf("expected ErrRefreshTokenConsumed, got: %v", err)
	}
}

func TestPaymentsWebhookIdempotentCancelsTicket(t *testing.T) {
	ctx, pool := connectDBOrSkip(t)
	var err error

	_, _ = pool.Exec(ctx, `
		TRUNCATE TABLE payments, tickets, events, refresh_tokens, users RESTART IDENTITY CASCADE
	`)

	userID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role)
		VALUES ($1, $2, 'x', 'student')
	`, userID, "user_" + uuid.NewString() + "@nu.edu.kz")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	eventID := uuid.New()
	capacityTotal := 1
	_, err = pool.Exec(ctx, `
		INSERT INTO events (id, title, description, starts_at, capacity_total, capacity_available)
		VALUES ($1, 'e', 'd', NOW(), $2, $2)
	`, eventID, capacityTotal)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	// Seat reservation via ticketing (capacity decrement).
	ticketRepo := ticketingRepo.NewPostgres(pool)
	ticketSvc := ticketingService.New(ticketRepo)
	ticket, _, err := ticketSvc.RegisterTicket(ctx, userID, eventID)
	if err != nil {
		t.Fatalf("register ticket: %v", err)
	}

	var capacityAvailable int
	if err := pool.QueryRow(ctx, `SELECT capacity_available FROM events WHERE id=$1`, eventID).Scan(&capacityAvailable); err != nil {
		t.Fatalf("query capacity_available: %v", err)
	}
	if capacityAvailable != 0 {
		t.Fatalf("expected capacity_available=0 after ticket reservation, got %d", capacityAvailable)
	}

	// Initiate payment (stub provider_url) and then fail via webhook twice.
	payRepo := paymentsRepo.NewPostgres(pool)
	paySvc := paymentsService.New(payRepo, ticketRepo)
	payment, _, err := paySvc.Initiate(ctx, userID, eventID, 100, "KZT")
	if err != nil {
		t.Fatalf("initiate payment: %v", err)
	}

	for i := 0; i < 2; i++ {
		_, err := paySvc.Webhook(ctx, payment.ProviderRef, paymentsModel.PaymentStatusFailed)
		if err != nil {
			t.Fatalf("webhook call %d failed: %v", i+1, err)
		}

		var ticketStatus string
		if err := pool.QueryRow(ctx, `
			SELECT status FROM tickets WHERE id=$1
		`, ticket.ID).Scan(&ticketStatus); err != nil {
			t.Fatalf("query ticket status: %v", err)
		}

		if ticketStatus != string(ticketingModel.TicketStatusCancelled) {
			t.Fatalf("expected ticket to be cancelled, got %q", ticketStatus)
		}

		if err := pool.QueryRow(ctx, `SELECT capacity_available FROM events WHERE id=$1`, eventID).Scan(&capacityAvailable); err != nil {
			t.Fatalf("query capacity_available: %v", err)
		}
		if capacityAvailable != capacityTotal {
			t.Fatalf("capacity should remain released after idempotent webhook calls, got %d", capacityAvailable)
		}
	}
}

