package ticketing_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"log/slog"

	apppkg "github.com/nu/student-event-ticketing-platform/internal/app"
	"github.com/nu/student-event-ticketing-platform/internal/config"
	"github.com/nu/student-event-ticketing-platform/internal/infra/db"
	"github.com/nu/student-event-ticketing-platform/internal/infra/redis"
)

// Defaults for host-side `go test`: 127.0.0.1, Postgres on 5433 (see docker-compose second port mapping), Redis on 6379.
// Override with POSTGRES_* / REDIS_* to match your environment (e.g. POSTGRES_PORT=5432 when nothing else uses that port).
func applyIntegrationHostDefaults(t *testing.T) {
	t.Helper()
	if os.Getenv("POSTGRES_HOST") == "" {
		t.Setenv("POSTGRES_HOST", "127.0.0.1")
	}
	if os.Getenv("POSTGRES_PORT") == "" {
		t.Setenv("POSTGRES_PORT", "5433")
	}
	if os.Getenv("REDIS_HOST") == "" {
		t.Setenv("REDIS_HOST", "127.0.0.1")
	}
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
		Role  string    `json:"role"`
	} `json:"user"`
}

type eventDTO struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	CoverImageURL     string    `json:"cover_image_url,omitempty"`
	StartsAt          time.Time `json:"starts_at"`
	CapacityTotal     int       `json:"capacity_total"`
	CapacityAvailable int       `json:"capacity_available"`
	Status            string    `json:"status"`
	ModerationStatus  string    `json:"moderation_status"`
}

type registerTicketResponse struct {
	TicketID    string `json:"ticket_id"`
	EventID     string `json:"event_id"`
	UserID      string `json:"user_id"`
	Status      string `json:"status"`
	QRPNGBase64 string `json:"qr_png_base64"`
	QRHashHex   string `json:"qr_hash_hex"`
}

type useTicketResponse struct {
	TicketID string `json:"ticket_id"`
	EventID  string `json:"event_id"`
	UserID   string `json:"user_id"`
	Status   string `json:"status"`
}

type myTicketsResponse struct {
	Tickets []struct {
		TicketID   string `json:"ticket_id"`
		Status     string `json:"status"`
		QRHashHex  string `json:"qr_hash_hex"`
		EventID    string `json:"event_id"`
		EventTitle string `json:"event_title"`
		EventDate  string `json:"event_date"`
	} `json:"tickets"`
}

var qrHashHexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// TestFullLifecycleTicketing exercises register → duplicate → capacity → QR check-in over the real HTTP router, Postgres, and Redis.
func TestFullLifecycleTicketing(t *testing.T) {
	applyIntegrationHostDefaults(t)
	ctx := context.Background()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Skipf("config: %v", err)
	}

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		t.Skipf("postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	var one int
	if err := pool.QueryRow(ctx, `SELECT 1 FROM events LIMIT 1`).Scan(&one); err != nil {
		t.Skipf("schema/events missing: %v", err)
	}
	var userRolesOK bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'user_roles'
		)
	`).Scan(&userRolesOK); err != nil || !userRolesOK {
		t.Skipf("user_roles table missing (apply docker/postgres/migrations/011_user_roles.sql): %v", err)
	}

	rdb, err := redis.Connect(ctx, cfg)
	if err != nil {
		t.Skipf("redis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	logger := slog.Default()
	srv := httptest.NewServer(apppkg.NewRouter(cfg, pool, rdb, logger, ctx))
	t.Cleanup(srv.Close)
	base := strings.TrimRight(srv.URL, "/")

	suffix := uuid.NewString()
	student1Email := "stu1_" + suffix + "@nu.edu.kz"
	student2Email := "stu2_" + suffix + "@nu.edu.kz"
	orgEmail := "org_" + suffix + "@nu.edu.kz"
	password := "TestPass1!long"

	var eventID uuid.UUID
	var student1ID, student2ID, orgID uuid.UUID

	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if eventID != (uuid.UUID{}) {
			_, _ = pool.Exec(cctx, `DELETE FROM tickets WHERE event_id = $1`, eventID)
			_, _ = pool.Exec(cctx, `DELETE FROM events WHERE id = $1`, eventID)
		}
		for _, id := range []uuid.UUID{student1ID, student2ID, orgID} {
			if id == (uuid.UUID{}) {
				continue
			}
			_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id = $1`, id)
		}
	})

	// --- Register users (API) ---
	a1 := mustRegister(t, base, student1Email, password)
	student1ID = a1.User.ID
	a2 := mustRegister(t, base, student2Email, password)
	student2ID = a2.User.ID
	aOrg := mustRegister(t, base, orgEmail, password)
	orgID = aOrg.User.ID

	_, err = pool.Exec(ctx, `UPDATE users SET role = 'organizer' WHERE id = $1`, orgID)
	if err != nil {
		t.Fatalf("promote organizer: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1 AND role = 'admin'`, orgID)
	if err != nil {
		t.Fatalf("sync user_roles (admin cleanup): %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role, status) VALUES ($1, 'student', 'active')
		ON CONFLICT (user_id, role) DO UPDATE SET status = 'active'
	`, orgID)
	if err != nil {
		t.Fatalf("sync user_roles (student): %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role, status) VALUES ($1, 'organizer', 'active')
		ON CONFLICT (user_id, role) DO UPDATE SET status = 'active'
	`, orgID)
	if err != nil {
		t.Fatalf("sync user_roles (organizer): %v", err)
	}

	orgLogin := mustLogin(t, base, orgEmail, password)
	if orgLogin.User.Role != "organizer" {
		t.Fatalf("expected organizer role after update, got %q", orgLogin.User.Role)
	}

	// --- Create event (organizer), capacity 1, starts in the future (registration open) ---
	starts := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	ev := mustCreateEvent(t, base, orgLogin.AccessToken, "lifecycle_"+suffix, "d", starts, 1)
	eventID = uuid.MustParse(ev.ID)
	if ev.ModerationStatus != "pending" && ev.ModerationStatus != "approved" {
		t.Logf("moderation_status=%q (continuing)", ev.ModerationStatus)
	}
	_, err = pool.Exec(ctx, `UPDATE events SET moderation_status = 'approved' WHERE id = $1`, eventID)
	if err != nil {
		t.Fatalf("approve event for ticketing: %v", err)
	}

	// Step 1: first registration
	reg1 := mustRegisterTicket(t, base, a1.AccessToken, eventID.String())
	if reg1.Status != "active" {
		t.Fatalf("ticket status: %q", reg1.Status)
	}
	if !qrHashHexRe.MatchString(reg1.QRHashHex) {
		t.Fatalf("invalid qr_hash_hex: %q", reg1.QRHashHex)
	}
	if reg1.QRPNGBase64 == "" {
		t.Fatal("expected qr_png_base64")
	}

	myRes := getJSONExpect(t, base, "/api/v1/tickets/my", a1.AccessToken, http.StatusOK)
	defer myRes.Body.Close()
	var myList myTicketsResponse
	if err := json.NewDecoder(myRes.Body).Decode(&myList); err != nil {
		t.Fatalf("decode my tickets: %v", err)
	}
	if len(myList.Tickets) != 1 {
		t.Fatalf("my tickets: want 1 item, got %d", len(myList.Tickets))
	}
	if myList.Tickets[0].TicketID != reg1.TicketID || myList.Tickets[0].EventID != eventID.String() {
		t.Fatalf("my tickets mismatch: %+v vs ticket %+v", myList.Tickets[0], reg1)
	}
	if myList.Tickets[0].QRHashHex != reg1.QRHashHex {
		t.Fatalf("my tickets qr_hash_hex: want %q got %q", reg1.QRHashHex, myList.Tickets[0].QRHashHex)
	}

	// Step 2: duplicate registration → 409
	postJSONExpect(t, base, "/api/v1/tickets/register", a1.AccessToken, map[string]any{
		"event_id": eventID.String(),
	}, http.StatusConflict)

	// Step 3: capacity exhausted → 409
	postJSONExpect(t, base, "/api/v1/tickets/register", a2.AccessToken, map[string]any{
		"event_id": eventID.String(),
	}, http.StatusConflict)

	// Step 4: check-in requires event to have started
	_, err = pool.Exec(ctx, `UPDATE events SET starts_at = NOW() - interval '2 hours' WHERE id = $1`, eventID)
	if err != nil {
		t.Fatalf("move event into past for check-in: %v", err)
	}

	useBody := map[string]any{"qr_hash_hex": reg1.QRHashHex}
	res := postJSON(t, base, "/api/v1/tickets/use", orgLogin.AccessToken, useBody)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("tickets/use: status %d", res.StatusCode)
	}
	var used useTicketResponse
	if err := json.NewDecoder(res.Body).Decode(&used); err != nil {
		t.Fatalf("decode use response: %v", err)
	}
	if used.Status != "used" {
		t.Fatalf("expected used ticket, got %q", used.Status)
	}
}

func mustRegister(t *testing.T, base, email, password string) authResponse {
	t.Helper()
	res := postJSON(t, base, "/api/v1/auth/register", "", map[string]any{
		"email": email, "password": password,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register %s: %d", email, res.StatusCode)
	}
	var out authResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	return out
}

func mustLogin(t *testing.T, base, email, password string) authResponse {
	t.Helper()
	res := postJSON(t, base, "/api/v1/auth/login", "", map[string]any{
		"email": email, "password": password,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login %s: %d", email, res.StatusCode)
	}
	var out authResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return out
}

func mustCreateEvent(t *testing.T, base, bearer, title, desc string, starts time.Time, cap int) eventDTO {
	t.Helper()
	res := postJSON(t, base, "/api/v1/events", bearer, map[string]any{
		"title": title, "description": desc, "starts_at": starts.Format(time.RFC3339Nano), "capacity_total": cap,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create event: %d", res.StatusCode)
	}
	var out eventDTO
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	return out
}

func mustRegisterTicket(t *testing.T, base, bearer, eventID string) registerTicketResponse {
	t.Helper()
	res := postJSON(t, base, "/api/v1/tickets/register", bearer, map[string]any{"event_id": eventID})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register ticket: %d", res.StatusCode)
	}
	var out registerTicketResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode ticket: %v", err)
	}
	return out
}

func postJSONExpect(t *testing.T, base, path, bearer string, body any, want int) {
	t.Helper()
	res := postJSON(t, base, path, bearer, body)
	defer res.Body.Close()
	if res.StatusCode != want {
		t.Fatalf("%s: want status %d, got %d", path, want, res.StatusCode)
	}
}

func postJSON(t *testing.T, base, path, bearer string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, base+path, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func getJSONExpect(t *testing.T, base, path, bearer string, want int) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, base+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != want {
		t.Fatalf("%s: want status %d, got %d", path, want, res.StatusCode)
	}
	return res
}
