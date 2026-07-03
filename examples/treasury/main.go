package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gapi-org/gapi"
	"github.com/gapi-org/gapi/middleware"
)

const (
	demoTenant   = "tnt_demo"
	ownerToken   = "sentinel_owner_token"
	analystToken = "sentinel_analyst_token"
)

type Role string

const (
	RoleOwner   Role = "owner"
	RoleAnalyst Role = "analyst"
)

type Principal struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email" format:"email"`
	Role     Role   `json:"role" enum:"owner,analyst"`
	TenantID string `json:"tenant_id"`
}

func (p Principal) canWrite() bool { return p.Role == RoleOwner }

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	CreatedAt time.Time `json:"created_at" format:"date-time"`
}

type Account struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name" validate:"required,min=2,max=80"`
	Currency       string    `json:"currency" validate:"required,len=3"`
	AvailableCents int       `json:"available_cents"`
	HeldCents      int       `json:"held_cents"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at" format:"date-time"`
}

type TransferStatus string

const (
	TransferPendingReview   TransferStatus = "pending_review"
	TransferPendingApproval TransferStatus = "pending_approval"
	TransferApproved        TransferStatus = "approved"
	TransferSettled         TransferStatus = "settled"
	TransferRejected        TransferStatus = "rejected"
)

type Transfer struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	SourceAccount  string         `json:"source_account"`
	DestAccount    string         `json:"dest_account"`
	AmountCents    int            `json:"amount_cents"`
	Currency       string         `json:"currency"`
	Status         TransferStatus `json:"status" enum:"pending_review,pending_approval,approved,settled,rejected"`
	RiskScore      int            `json:"risk_score"`
	Reason         string         `json:"reason,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time      `json:"created_at" format:"date-time"`
	UpdatedAt      time.Time      `json:"updated_at" format:"date-time"`
}

type LedgerEntry struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	TransferID  string    `json:"transfer_id"`
	AccountID   string    `json:"account_id"`
	Direction   string    `json:"direction" enum:"debit,credit"`
	AmountCents int       `json:"amount_cents"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at" format:"date-time"`
}

type AuditEvent struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	ActorID   string         `json:"actor_id"`
	Type      string         `json:"type"`
	Resource  string         `json:"resource"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at" format:"date-time"`
}

type PlatformEvent struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at" format:"date-time"`
}

type PageMeta struct {
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type Store struct {
	mu           sync.Mutex
	nextAccount  int
	nextTransfer int
	nextLedger   int
	nextAudit    int
	nextEvent    int
	tenants      map[string]Tenant
	tokens       map[string]Principal
	accounts     map[string]Account
	transfers    map[string]Transfer
	idempotency  map[string]string
	ledger       []LedgerEntry
	audit        []AuditEvent
	events       []PlatformEvent
}

func newStore() *Store {
	now := time.Now().UTC()
	s := &Store{
		tenants: map[string]Tenant{demoTenant: {ID: demoTenant, Name: "Sentinel Treasury", Region: "us-east-1", CreatedAt: now}},
		tokens: map[string]Principal{
			ownerToken:   {UserID: "usr_owner", Email: "owner@sentinel.dev", Role: RoleOwner, TenantID: demoTenant},
			analystToken: {UserID: "usr_analyst", Email: "analyst@sentinel.dev", Role: RoleAnalyst, TenantID: demoTenant},
		},
		accounts:    map[string]Account{},
		transfers:   map[string]Transfer{},
		idempotency: map[string]string{},
	}
	s.seedAccount("acct_operating", "Operating", "USD", 2_000_000)
	s.seedAccount("acct_reserve", "Reserve", "USD", 500_000)
	return s
}

func (s *Store) seedAccount(id, name, currency string, balance int) {
	now := time.Now().UTC()
	s.accounts[id] = Account{ID: id, TenantID: demoTenant, Name: name, Currency: currency, AvailableCents: balance, Active: true, CreatedAt: now}
}

func (s *Store) authenticate(token string) (Principal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.tokens[token]
	return p, ok
}

func (s *Store) emit(tenantID, actorID, typ, resource string, payload map[string]any) {
	s.nextAudit++
	s.nextEvent++
	now := time.Now().UTC()
	s.audit = append(s.audit, AuditEvent{ID: fmt.Sprintf("aud_%06d", s.nextAudit), TenantID: tenantID, ActorID: actorID, Type: typ, Resource: resource, Payload: payload, CreatedAt: now})
	s.events = append(s.events, PlatformEvent{ID: fmt.Sprintf("evt_%06d", s.nextEvent), TenantID: tenantID, Type: typ, Payload: payload, CreatedAt: now})
}

type API struct {
	store *Store
}

func authDep(store *Store) gapi.Dependency[Principal] {
	return gapi.Dep("principal", func(ctx context.Context, r *http.Request) (Principal, error) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			return Principal{}, gapi.NewHTTPError(http.StatusUnauthorized, "Missing bearer token.")
		}
		principal, ok := store.authenticate(token)
		if !ok {
			return Principal{}, gapi.NewHTTPError(http.StatusUnauthorized, "Invalid bearer token.")
		}
		return principal, nil
	})
}

func ensureTenant(principal Principal, tenantID string) error {
	if principal.TenantID != tenantID {
		return gapi.NewHTTPError(http.StatusForbidden, "Cross-tenant access denied.")
	}
	return nil
}

func ensureOwner(principal Principal) error {
	if !principal.canWrite() {
		return gapi.NewHTTPError(http.StatusForbidden, "Owner role required.")
	}
	return nil
}

type HealthOut struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func (api *API) Health(ctx context.Context, in struct{}) (HealthOut, error) {
	return HealthOut{Status: "ok", Service: "sentinel"}, nil
}

type MeIn struct {
	Principal Principal `dep:"principal"`
}

type MeOut struct {
	Principal Principal `json:"principal"`
	Tenant    Tenant    `json:"tenant"`
}

func (api *API) Me(ctx context.Context, in MeIn) (MeOut, error) {
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	return MeOut{Principal: in.Principal, Tenant: api.store.tenants[in.Principal.TenantID]}, nil
}

type TenantPathIn struct {
	TenantID  string    `path:"tenant_id" validate:"required"`
	Principal Principal `dep:"principal"`
}

type ListAccountsIn struct {
	TenantID   string    `path:"tenant_id" validate:"required"`
	ActiveOnly bool      `query:"active_only" default:"true"`
	Limit      int       `query:"limit" validate:"min=1,max=100" default:"25"`
	Principal  Principal `dep:"principal"`
}

type ListAccountsOut struct {
	Accounts []Account `json:"accounts"`
	Meta     PageMeta  `json:"meta"`
}

func (api *API) ListAccounts(ctx context.Context, in ListAccountsIn) (ListAccountsOut, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return ListAccountsOut{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	accounts := make([]Account, 0)
	for _, account := range api.store.accounts {
		if account.TenantID == in.TenantID && (!in.ActiveOnly || account.Active) {
			accounts = append(accounts, account)
		}
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
	if in.Limit == 0 {
		in.Limit = 25
	}
	total := len(accounts)
	if len(accounts) > in.Limit {
		accounts = accounts[:in.Limit]
	}
	return ListAccountsOut{Accounts: accounts, Meta: PageMeta{Limit: in.Limit, Total: total}}, nil
}

type CreateAccountIn struct {
	TenantID  string    `path:"tenant_id" validate:"required"`
	Principal Principal `dep:"principal"`
	Body      struct {
		Name                string `json:"name" validate:"required,min=2,max=80"`
		Currency            string `json:"currency" validate:"required,len=3"`
		OpeningBalanceCents int    `json:"opening_balance_cents" validate:"min=0"`
	} `body:""`
}

func (api *API) CreateAccount(ctx context.Context, in CreateAccountIn) (gapi.Response[Account], error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return gapi.Response[Account]{}, err
	}
	if err := ensureOwner(in.Principal); err != nil {
		return gapi.Response[Account]{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	api.store.nextAccount++
	now := time.Now().UTC()
	account := Account{ID: fmt.Sprintf("acct_%06d", api.store.nextAccount), TenantID: in.TenantID, Name: in.Body.Name, Currency: in.Body.Currency, AvailableCents: in.Body.OpeningBalanceCents, Active: true, CreatedAt: now}
	api.store.accounts[account.ID] = account
	api.store.emit(in.TenantID, in.Principal.UserID, "account.created", account.ID, map[string]any{"name": account.Name})
	return gapi.Response[Account]{Status: http.StatusCreated, Headers: http.Header{"Location": []string{fmt.Sprintf("/api/v1/tenants/%s/accounts/%s", in.TenantID, account.ID)}}, Body: account}, nil
}

type AccountPathIn struct {
	TenantID  string    `path:"tenant_id" validate:"required"`
	AccountID string    `path:"account_id" validate:"required"`
	Principal Principal `dep:"principal"`
}

func (api *API) GetAccount(ctx context.Context, in AccountPathIn) (Account, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return Account{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	account, ok := api.store.accounts[in.AccountID]
	if !ok || account.TenantID != in.TenantID {
		return Account{}, gapi.NewHTTPError(http.StatusNotFound, "Account not found.")
	}
	return account, nil
}

type CreateTransferIn struct {
	TenantID       string    `path:"tenant_id" validate:"required"`
	IdempotencyKey string    `header:"Idempotency-Key" validate:"required,min=8,max=128"`
	Principal      Principal `dep:"principal"`
	Body           struct {
		SourceAccount string `json:"source_account" validate:"required"`
		DestAccount   string `json:"dest_account" validate:"required"`
		AmountCents   int    `json:"amount_cents" validate:"min=1,max=100000000"`
		Currency      string `json:"currency" validate:"required,len=3"`
		Reason        string `json:"reason" validate:"max=160"`
	} `body:""`
}

func riskScore(amount int, reason string) int {
	score := amount / 5000
	if strings.Contains(strings.ToLower(reason), "external") {
		score += 35
	}
	if score > 100 {
		return 100
	}
	return score
}

func (api *API) CreateTransfer(ctx context.Context, in CreateTransferIn) (gapi.Response[Transfer], error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return gapi.Response[Transfer]{}, err
	}
	if err := ensureOwner(in.Principal); err != nil {
		return gapi.Response[Transfer]{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	if id, ok := api.store.idempotency[in.TenantID+"|"+in.IdempotencyKey]; ok {
		return gapi.Response[Transfer]{Status: http.StatusOK, Body: api.store.transfers[id]}, nil
	}
	source, ok := api.store.accounts[in.Body.SourceAccount]
	if !ok || source.TenantID != in.TenantID || !source.Active {
		return gapi.Response[Transfer]{}, gapi.NewHTTPError(http.StatusBadRequest, "Invalid source account.")
	}
	dest, ok := api.store.accounts[in.Body.DestAccount]
	if !ok || dest.TenantID != in.TenantID || !dest.Active {
		return gapi.Response[Transfer]{}, gapi.NewHTTPError(http.StatusBadRequest, "Invalid destination account.")
	}
	if source.Currency != in.Body.Currency || dest.Currency != in.Body.Currency {
		return gapi.Response[Transfer]{}, gapi.NewHTTPError(http.StatusConflict, "Currency mismatch.")
	}
	if source.AvailableCents < in.Body.AmountCents {
		return gapi.Response[Transfer]{}, gapi.NewHTTPError(http.StatusConflict, "Insufficient available balance.")
	}
	source.AvailableCents -= in.Body.AmountCents
	source.HeldCents += in.Body.AmountCents
	api.store.accounts[source.ID] = source

	score := riskScore(in.Body.AmountCents, in.Body.Reason)
	status := TransferPendingApproval
	if score >= 70 {
		status = TransferPendingReview
	}
	api.store.nextTransfer++
	now := time.Now().UTC()
	transfer := Transfer{ID: fmt.Sprintf("trf_%06d", api.store.nextTransfer), TenantID: in.TenantID, SourceAccount: source.ID, DestAccount: dest.ID, AmountCents: in.Body.AmountCents, Currency: in.Body.Currency, Status: status, RiskScore: score, Reason: in.Body.Reason, IdempotencyKey: in.IdempotencyKey, CreatedAt: now, UpdatedAt: now}
	api.store.transfers[transfer.ID] = transfer
	api.store.idempotency[in.TenantID+"|"+in.IdempotencyKey] = transfer.ID
	api.store.emit(in.TenantID, in.Principal.UserID, "transfer.created", transfer.ID, map[string]any{"status": transfer.Status, "risk_score": transfer.RiskScore})
	return gapi.Response[Transfer]{Status: http.StatusCreated, Headers: http.Header{"Location": []string{fmt.Sprintf("/api/v1/tenants/%s/transfers/%s", in.TenantID, transfer.ID)}}, Body: transfer}, nil
}

type ListTransfersIn struct {
	TenantID  string    `path:"tenant_id" validate:"required"`
	Status    string    `query:"status" enum:"pending_review,pending_approval,approved,settled,rejected"`
	Principal Principal `dep:"principal"`
}

type ListTransfersOut struct {
	Transfers []Transfer `json:"transfers"`
	Meta      PageMeta   `json:"meta"`
}

func (api *API) ListTransfers(ctx context.Context, in ListTransfersIn) (ListTransfersOut, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return ListTransfersOut{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	transfers := make([]Transfer, 0)
	for _, transfer := range api.store.transfers {
		if transfer.TenantID == in.TenantID && (in.Status == "" || string(transfer.Status) == in.Status) {
			transfers = append(transfers, transfer)
		}
	}
	sort.Slice(transfers, func(i, j int) bool { return transfers[i].CreatedAt.After(transfers[j].CreatedAt) })
	return ListTransfersOut{Transfers: transfers, Meta: PageMeta{Limit: len(transfers), Total: len(transfers)}}, nil
}

type TransferPathIn struct {
	TenantID   string    `path:"tenant_id" validate:"required"`
	TransferID string    `path:"transfer_id" validate:"required"`
	Principal  Principal `dep:"principal"`
}

func (api *API) GetTransfer(ctx context.Context, in TransferPathIn) (Transfer, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return Transfer{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	transfer, ok := api.store.transfers[in.TransferID]
	if !ok || transfer.TenantID != in.TenantID {
		return Transfer{}, gapi.NewHTTPError(http.StatusNotFound, "Transfer not found.")
	}
	return transfer, nil
}

type DecideTransferIn struct {
	TenantID   string    `path:"tenant_id" validate:"required"`
	TransferID string    `path:"transfer_id" validate:"required"`
	Principal  Principal `dep:"principal"`
	Body       struct {
		Decision string `json:"decision" validate:"required" enum:"approve,settle,reject"`
		Reason   string `json:"reason" validate:"max=160"`
	} `body:""`
}

func (api *API) DecideTransfer(ctx context.Context, in DecideTransferIn) (Transfer, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return Transfer{}, err
	}
	if err := ensureOwner(in.Principal); err != nil {
		return Transfer{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	transfer, ok := api.store.transfers[in.TransferID]
	if !ok || transfer.TenantID != in.TenantID {
		return Transfer{}, gapi.NewHTTPError(http.StatusNotFound, "Transfer not found.")
	}
	source := api.store.accounts[transfer.SourceAccount]
	dest := api.store.accounts[transfer.DestAccount]
	switch in.Body.Decision {
	case "approve":
		if transfer.Status != TransferPendingReview && transfer.Status != TransferPendingApproval {
			return Transfer{}, gapi.NewHTTPError(http.StatusConflict, "Transfer cannot be approved from current state.")
		}
		transfer.Status = TransferApproved
	case "settle":
		if transfer.Status != TransferApproved {
			return Transfer{}, gapi.NewHTTPError(http.StatusConflict, "Only approved transfers can settle.")
		}
		source.HeldCents -= transfer.AmountCents
		dest.AvailableCents += transfer.AmountCents
		api.store.accounts[source.ID] = source
		api.store.accounts[dest.ID] = dest
		api.store.nextLedger++
		api.store.ledger = append(api.store.ledger, LedgerEntry{ID: fmt.Sprintf("led_%06d", api.store.nextLedger), TenantID: in.TenantID, TransferID: transfer.ID, AccountID: source.ID, Direction: "debit", AmountCents: transfer.AmountCents, Currency: transfer.Currency, CreatedAt: time.Now().UTC()})
		api.store.nextLedger++
		api.store.ledger = append(api.store.ledger, LedgerEntry{ID: fmt.Sprintf("led_%06d", api.store.nextLedger), TenantID: in.TenantID, TransferID: transfer.ID, AccountID: dest.ID, Direction: "credit", AmountCents: transfer.AmountCents, Currency: transfer.Currency, CreatedAt: time.Now().UTC()})
		transfer.Status = TransferSettled
	case "reject":
		if transfer.Status != TransferPendingReview && transfer.Status != TransferPendingApproval && transfer.Status != TransferApproved {
			return Transfer{}, gapi.NewHTTPError(http.StatusConflict, "Transfer cannot be rejected from current state.")
		}
		source.HeldCents -= transfer.AmountCents
		source.AvailableCents += transfer.AmountCents
		api.store.accounts[source.ID] = source
		transfer.Status = TransferRejected
	}
	transfer.Reason = in.Body.Reason
	transfer.UpdatedAt = time.Now().UTC()
	api.store.transfers[transfer.ID] = transfer
	api.store.emit(in.TenantID, in.Principal.UserID, "transfer."+in.Body.Decision, transfer.ID, map[string]any{"status": transfer.Status})
	return transfer, nil
}

type ListLedgerOut struct {
	Entries []LedgerEntry `json:"entries"`
	Meta    PageMeta      `json:"meta"`
}

func (api *API) ListLedger(ctx context.Context, in TenantPathIn) (ListLedgerOut, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return ListLedgerOut{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	entries := make([]LedgerEntry, 0)
	for _, entry := range api.store.ledger {
		if entry.TenantID == in.TenantID {
			entries = append(entries, entry)
		}
	}
	return ListLedgerOut{Entries: entries, Meta: PageMeta{Limit: len(entries), Total: len(entries)}}, nil
}

type AuditOut struct {
	Events []AuditEvent `json:"events"`
	Meta   PageMeta     `json:"meta"`
}

func (api *API) Audit(ctx context.Context, in TenantPathIn) (AuditOut, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return AuditOut{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	events := make([]AuditEvent, 0)
	for i := len(api.store.audit) - 1; i >= 0; i-- {
		if api.store.audit[i].TenantID == in.TenantID {
			events = append(events, api.store.audit[i])
		}
	}
	return AuditOut{Events: events, Meta: PageMeta{Limit: len(events), Total: len(events)}}, nil
}

type ExposureOut struct {
	TenantID          string `json:"tenant_id"`
	AvailableCents    int    `json:"available_cents"`
	HeldCents         int    `json:"held_cents"`
	SettledCents      int    `json:"settled_cents"`
	RejectedTransfers int    `json:"rejected_transfers"`
	OpenTransfers     int    `json:"open_transfers"`
}

func (api *API) Exposure(ctx context.Context, in TenantPathIn) (ExposureOut, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return ExposureOut{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	out := ExposureOut{TenantID: in.TenantID}
	for _, account := range api.store.accounts {
		if account.TenantID == in.TenantID {
			out.AvailableCents += account.AvailableCents
			out.HeldCents += account.HeldCents
		}
	}
	for _, transfer := range api.store.transfers {
		if transfer.TenantID != in.TenantID {
			continue
		}
		switch transfer.Status {
		case TransferSettled:
			out.SettledCents += transfer.AmountCents
		case TransferRejected:
			out.RejectedTransfers++
		default:
			out.OpenTransfers++
		}
	}
	return out, nil
}

func (api *API) LedgerCSV(ctx context.Context, in TenantPathIn) (gapi.Text, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return gapi.Text{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	var b strings.Builder
	b.WriteString("id,transfer_id,account_id,direction,amount_cents,currency,created_at\n")
	for _, entry := range api.store.ledger {
		if entry.TenantID == in.TenantID {
			b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d,%s,%s\n", entry.ID, entry.TransferID, entry.AccountID, entry.Direction, entry.AmountCents, entry.Currency, entry.CreatedAt.Format(time.RFC3339)))
		}
	}
	return gapi.Text{Status: http.StatusOK, Headers: http.Header{"Content-Disposition": []string{`attachment; filename="ledger.csv"`}}, Body: b.String()}, nil
}

type StreamEventsIn struct {
	TenantID  string    `path:"tenant_id" validate:"required"`
	Principal Principal `dep:"principal"`
}

func (api *API) Events(ctx context.Context, in StreamEventsIn) (gapi.SSE, error) {
	if err := ensureTenant(in.Principal, in.TenantID); err != nil {
		return gapi.SSE{}, err
	}
	api.store.mu.Lock()
	defer api.store.mu.Unlock()
	events := []gapi.SSEEvent{{Event: "connected", Data: fmt.Sprintf(`{"tenant_id":%q}`, in.TenantID)}}
	for _, event := range api.store.events {
		if event.TenantID == in.TenantID {
			payload, _ := json.Marshal(event)
			events = append(events, gapi.SSEEvent{Event: event.Type, ID: event.ID, Data: string(payload)})
		}
	}
	return gapi.SSE{Events: events}, nil
}

func newApp() *gapi.App {
	store := newStore()
	api := &API{store: store}
	app := gapi.New(gapi.Config{Title: "Sentinel Treasury API", Version: "0.1.0"})
	app.RegisterSecurity(gapi.BearerAuth("bearerAuth"))
	app.Use(middleware.Recover())
	app.Use(middleware.RequestID())
	app.Use(middleware.BodyLimit(1 << 20))
	app.Use(middleware.Timeout(10 * time.Second))
	app.Provide(authDep(store))

	gapi.Get(app, "/health", api.Health, gapi.Summary("Health check"))
	v1 := app.Group("/api/v1")
	v1.Use(middleware.SecureHeaders())
	gapi.Get(v1, "/me", api.Me, gapi.Summary("Current principal"), gapi.Tags("auth"), gapi.Security("bearerAuth"))

	tenants := v1.Group("/tenants/{tenant_id}")
	gapi.Get(tenants, "/accounts", api.ListAccounts, gapi.Summary("List accounts"), gapi.Tags("accounts"), gapi.Security("bearerAuth"))
	gapi.Post(tenants, "/accounts", api.CreateAccount, gapi.Summary("Create account"), gapi.Tags("accounts"), gapi.Security("bearerAuth"))
	gapi.Get(tenants, "/accounts/{account_id}", api.GetAccount, gapi.Summary("Get account"), gapi.Tags("accounts"), gapi.Security("bearerAuth"))
	gapi.Get(tenants, "/transfers", api.ListTransfers, gapi.Summary("List transfers"), gapi.Tags("transfers"), gapi.Security("bearerAuth"))
	gapi.Post(tenants, "/transfers", api.CreateTransfer, gapi.Summary("Create transfer"), gapi.Tags("transfers"), gapi.Security("bearerAuth"))
	gapi.Get(tenants, "/transfers/{transfer_id}", api.GetTransfer, gapi.Summary("Get transfer"), gapi.Tags("transfers"), gapi.Security("bearerAuth"))
	gapi.Post(tenants, "/transfers/{transfer_id}/decisions", api.DecideTransfer, gapi.Summary("Approve, settle, or reject transfer"), gapi.Tags("transfers"), gapi.Security("bearerAuth"))
	gapi.Get(tenants, "/ledger", api.ListLedger, gapi.Summary("List ledger entries"), gapi.Tags("ledger"), gapi.Security("bearerAuth"))
	gapi.Get(tenants, "/audit", api.Audit, gapi.Summary("Audit trail"), gapi.Tags("audit"), gapi.Security("bearerAuth"))
	gapi.Get(tenants, "/reports/exposure", api.Exposure, gapi.Summary("Treasury exposure report"), gapi.Tags("reports"), gapi.Security("bearerAuth"))
	gapi.Get(tenants, "/reports/ledger.csv", api.LedgerCSV, gapi.Summary("Export ledger as CSV"), gapi.Tags("reports"), gapi.Security("bearerAuth"))
	gapi.Get(tenants, "/events/stream", api.Events, gapi.Summary("Stream platform events"), gapi.Tags("events"), gapi.Security("bearerAuth"))
	return app
}

func main() {
	http.ListenAndServe(":8080", newApp())
}
