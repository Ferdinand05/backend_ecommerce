package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"ferdinand/ecommerce/internal/models"
	xenditclient "ferdinand/ecommerce/internal/payment/xendit"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

// --- fakes ---

type fakeOrderRepo struct {
	order models.Order
	err   error

	histories     []models.OrderStatusHistory
	updatedStatus string
}

func (f *fakeOrderRepo) FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error) {
	return f.order, f.err
}

func (f *fakeOrderRepo) FindByIDForUpdate(ctx context.Context, orderID uuid.UUID) (models.Order, error) {
	return f.order, f.err
}

func (f *fakeOrderRepo) UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error {
	f.updatedStatus = status
	f.order.Status = status
	return nil
}

func (f *fakeOrderRepo) CreateStatusHistory(ctx context.Context, history *models.OrderStatusHistory) error {
	f.histories = append(f.histories, *history)
	return nil
}

type statusUpdate struct {
	paymentID uuid.UUID
	status    string
}

type fakePaymentRepo struct {
	payments  map[uuid.UUID]models.Payment
	createErr error
	latest    models.Payment
	latestErr error

	created        []*models.Payment
	statusUpdates  []statusUpdate
	invoiceUpdates []*models.Payment
}

func (f *fakePaymentRepo) FindByIDForUpdate(ctx context.Context, paymentID uuid.UUID) (models.Payment, error) {
	return f.FindByID(ctx, paymentID)
}

func (f *fakePaymentRepo) MarkPaid(ctx context.Context, paymentID uuid.UUID, paidAt *time.Time) (bool, error) {
	p, ok := f.payments[paymentID]
	if !ok {
		return false, ErrPaymentNotFound
	}
	if p.Status == string(PaymentStatusPaid) {
		return false, nil
	}
	p.Status = string(PaymentStatusPaid)
	p.PaidAt = paidAt
	f.payments[paymentID] = p
	f.statusUpdates = append(f.statusUpdates, statusUpdate{paymentID: paymentID, status: string(PaymentStatusPaid)})
	return true, nil
}

func (f *fakePaymentRepo) MarkExpiredPending(ctx context.Context, paymentID uuid.UUID) (bool, error) {
	p, ok := f.payments[paymentID]
	if !ok {
		return false, ErrPaymentNotFound
	}
	if p.Status != string(PaymentStatusPending) {
		return false, nil
	}
	p.Status = string(PaymentStatusExpired)
	f.payments[paymentID] = p
	f.statusUpdates = append(f.statusUpdates, statusUpdate{paymentID: paymentID, status: string(PaymentStatusExpired)})
	return true, nil
}

func (f *fakePaymentRepo) FailOtherPendingByOrderID(ctx context.Context, orderID uuid.UUID, excludePaymentID uuid.UUID) error {
	for id, p := range f.payments {
		if p.OrderID == orderID && p.Status == string(PaymentStatusPending) && id != excludePaymentID {
			p.Status = string(PaymentStatusFailed)
			f.payments[id] = p
		}
	}
	return nil
}

func newFakePaymentRepo() *fakePaymentRepo {
	return &fakePaymentRepo{payments: make(map[uuid.UUID]models.Payment)}
}

func (f *fakePaymentRepo) Create(ctx context.Context, payment *models.Payment) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, payment)
	f.payments[payment.ID] = *payment
	return nil
}

func (f *fakePaymentRepo) FindByID(ctx context.Context, paymentID uuid.UUID) (models.Payment, error) {
	p, ok := f.payments[paymentID]
	if !ok {
		return models.Payment{}, ErrPaymentNotFound
	}
	return p, nil
}

func (f *fakePaymentRepo) FindByExternalID(ctx context.Context, externalID string) (models.Payment, error) {
	for _, p := range f.payments {
		if p.ExternalID == externalID {
			return p, nil
		}
	}
	return models.Payment{}, ErrPaymentNotFound
}

func (f *fakePaymentRepo) FindByProviderInvoiceID(ctx context.Context, invoiceID string) (models.Payment, error) {
	for _, p := range f.payments {
		if p.ProviderInvoiceID != nil && *p.ProviderInvoiceID == invoiceID {
			return p, nil
		}
	}
	return models.Payment{}, ErrPaymentNotFound
}

func (f *fakePaymentRepo) FindLatestByOrderID(ctx context.Context, orderID uuid.UUID) (models.Payment, error) {
	if f.latestErr != nil {
		return models.Payment{}, f.latestErr
	}
	return f.latest, nil
}

func (f *fakePaymentRepo) UpdateInvoice(
	ctx context.Context,
	paymentID uuid.UUID,
	invoiceID string,
	checkoutURL string,
	expiresAt *time.Time,
) error {
	p, ok := f.payments[paymentID]
	if !ok {
		return ErrPaymentNotFound
	}
	p.ProviderInvoiceID = &invoiceID
	p.CheckoutURL = &checkoutURL
	p.ExpiresAt = expiresAt
	f.payments[paymentID] = p
	f.invoiceUpdates = append(f.invoiceUpdates, &p)
	return nil
}

func (f *fakePaymentRepo) UpdateStatus(
	ctx context.Context,
	paymentID uuid.UUID,
	status string,
	paidAt *time.Time,
) error {
	p, ok := f.payments[paymentID]
	if !ok {
		return ErrPaymentNotFound
	}
	p.Status = status
	f.payments[paymentID] = p
	f.statusUpdates = append(f.statusUpdates, statusUpdate{paymentID: paymentID, status: status})
	return nil
}

type fakeXendit struct {
	createRes *xenditclient.InvoiceResult
	createErr error

	found    *xenditclient.InvoiceResult
	foundErr error

	createCalls int
	foundCalls  int
}

func (f *fakeXendit) CreateInvoice(
	ctx context.Context,
	req xenditclient.CreateInvoiceRequest,
) (xenditclient.InvoiceResult, error) {
	f.createCalls++
	if f.createErr != nil {
		return xenditclient.InvoiceResult{}, f.createErr
	}
	return *f.createRes, nil
}

func (f *fakeXendit) FindLatestInvoiceByExternalID(
	ctx context.Context,
	externalID string,
) (*xenditclient.InvoiceResult, error) {
	f.foundCalls++
	return f.found, f.foundErr
}

// --- helpers ---

func payableOrder(userID uuid.UUID) models.Order {
	expiry := time.Now().Add(20 * time.Minute)
	return models.Order{
		ID:               uuid.New(),
		UserID:           userID,
		OrderNumber:      "ORD-20260101-ABCD1234",
		Status:           "PENDING",
		TotalAmount:      decimal.NewFromInt(120000),
		PaymentExpiresAt: &expiry,
	}
}

func invoiceResult(status string) xenditclient.InvoiceResult {
	expiry := time.Now().Add(15 * time.Minute)
	return xenditclient.InvoiceResult{
		ID:          "inv-" + uuid.NewString(),
		ExternalID:  "PAY-" + uuid.NewString(),
		Status:      status,
		CheckoutURL: "https://checkout.example/inv",
		ExpiryDate:  &expiry,
	}
}

func newTestService(ord *models.Order, repo *fakePaymentRepo, xendit *fakeXendit) Service {
	orderRepo := &fakeOrderRepo{}
	if ord != nil {
		orderRepo.order = *ord
	}
	return NewService(nil, orderRepo, repo, nil, nil, xendit)
}

func newPendingPayment(externalID string) models.Payment {
	return models.Payment{
		ID:         uuid.New(),
		OrderID:    uuid.New(),
		ExternalID: externalID,
		Status:     string(PaymentStatusPending),
		Amount:     decimal.NewFromInt(120000),
		Currency:   "IDR",
	}
}

// --- tests ---

func TestCreatePaymentSuccess(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)
	repo := newFakePaymentRepo()
	xendit := &fakeXendit{createRes: ptr(invoiceResult(xenditclient.InvoiceStatusPending))}

	svc := newTestService(&ord, repo, xendit)

	result, err := svc.CreatePayment(context.Background(), userID, ord.ID)
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}

	if len(repo.created) != 1 {
		t.Fatalf("created = %d, want 1", len(repo.created))
	}
	if result.CheckoutURL == "" {
		t.Fatal("checkout_url kosong")
	}
	if result.Status != string(PaymentStatusPending) {
		t.Fatalf("status = %q, want PENDING", result.Status)
	}
	if result.Amount != "120000" {
		t.Fatalf("amount = %q, want 120000", result.Amount)
	}
	if xendit.createCalls != 1 {
		t.Fatalf("CreateInvoice calls = %d, want 1", xendit.createCalls)
	}
}

func TestCreatePaymentReusesValidPendingInvoice(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)

	pending := newPendingPayment("PAY-EXISTING")
	url := "https://checkout.example/live"
	expiry := time.Now().Add(10 * time.Minute)
	pending.CheckoutURL = &url
	pending.ExpiresAt = &expiry

	repo := newFakePaymentRepo()
	repo.latest = pending

	xendit := &fakeXendit{}
	svc := newTestService(&ord, repo, xendit)

	result, err := svc.CreatePayment(context.Background(), userID, ord.ID)
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}

	if result.CheckoutURL != url {
		t.Fatalf("checkout_url = %q, want reuse %q", result.CheckoutURL, url)
	}
	if xendit.createCalls != 0 || xendit.foundCalls != 0 {
		t.Fatalf("xendit dipanggil: create=%d found=%d, want 0/0", xendit.createCalls, xendit.foundCalls)
	}
	if len(repo.created) != 0 {
		t.Fatalf("attempt baru dibuat = %d, want 0", len(repo.created))
	}
}

func TestCreatePaymentOwnershipAndState(t *testing.T) {
	userID := uuid.New()

	// Order milik user lain.
	ord := payableOrder(uuid.New())
	svc := newTestService(&ord, newFakePaymentRepo(), &fakeXendit{})
	if _, err := svc.CreatePayment(context.Background(), userID, ord.ID); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("err = %v, want ErrOrderNotFound", err)
	}

	// Order bukan PENDING.
	ord = payableOrder(userID)
	ord.Status = "PAID"
	svc = newTestService(&ord, newFakePaymentRepo(), &fakeXendit{})
	if _, err := svc.CreatePayment(context.Background(), userID, ord.ID); !errors.Is(err, ErrOrderNotPayable) {
		t.Fatalf("err = %v, want ErrOrderNotPayable", err)
	}

	// Periode bayar habis.
	ord = payableOrder(userID)
	expired := time.Now().Add(-time.Minute)
	ord.PaymentExpiresAt = &expired
	svc = newTestService(&ord, newFakePaymentRepo(), &fakeXendit{})
	if _, err := svc.CreatePayment(context.Background(), userID, ord.ID); !errors.Is(err, ErrOrderPaymentExpired) {
		t.Fatalf("err = %v, want ErrOrderPaymentExpired", err)
	}
}

func TestCreatePaymentRecoversPendingWithoutURL(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)

	// PENDING tanpa URL: UpdateInvoice sempat gagal sebelumnya.
	pending := newPendingPayment("PAY-BROKEN")
	repo := newFakePaymentRepo()
	repo.latest = pending
	repo.payments[pending.ID] = pending

	live := invoiceResult(xenditclient.InvoiceStatusPending)
	live.ExternalID = "PAY-BROKEN"
	xendit := &fakeXendit{found: &live}

	svc := newTestService(&ord, repo, xendit)

	result, err := svc.CreatePayment(context.Background(), userID, ord.ID)
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}

	if xendit.createCalls != 0 {
		t.Fatalf("CreateInvoice dipanggil %d kali, want 0 (harus recover)", xendit.createCalls)
	}
	if len(repo.invoiceUpdates) != 1 {
		t.Fatalf("UpdateInvoice calls = %d, want 1", len(repo.invoiceUpdates))
	}
	if result.CheckoutURL != live.CheckoutURL {
		t.Fatalf("checkout_url = %q, want %q", result.CheckoutURL, live.CheckoutURL)
	}
}

func TestCreatePaymentExpiredInvoiceCreatesNewAttempt(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)

	pending := newPendingPayment("PAY-OLD")
	repo := newFakePaymentRepo()
	repo.latest = pending
	repo.payments[pending.ID] = pending

	expired := invoiceResult(xenditclient.InvoiceStatusExpired)
	expired.ExternalID = "PAY-OLD"
	live := invoiceResult(xenditclient.InvoiceStatusPending)
	xendit := &fakeXendit{found: &expired, createRes: &live}

	svc := newTestService(&ord, repo, xendit)

	result, err := svc.CreatePayment(context.Background(), userID, ord.ID)
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}

	if len(repo.statusUpdates) != 1 || repo.statusUpdates[0].status != string(PaymentStatusFailed) {
		t.Fatalf("status updates = %+v, want satu FAILED", repo.statusUpdates)
	}
	if len(repo.created) != 1 {
		t.Fatalf("attempt baru = %d, want 1", len(repo.created))
	}
	if result.Status != string(PaymentStatusPending) || result.CheckoutURL == "" {
		t.Fatalf("result = %+v, want invoice PENDING baru", result)
	}
}

func TestCreatePaymentDetectsPaidInvoice(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)

	pending := newPendingPayment("PAY-PAID")
	repo := newFakePaymentRepo()
	repo.latest = pending
	repo.payments[pending.ID] = pending

	paid := invoiceResult(xenditclient.InvoiceStatusPaid)
	paid.ExternalID = "PAY-PAID"
	xendit := &fakeXendit{found: &paid}

	svc := newTestService(&ord, repo, xendit)

	if _, err := svc.CreatePayment(context.Background(), userID, ord.ID); !errors.Is(err, ErrOrderNotPayable) {
		t.Fatalf("err = %v, want ErrOrderNotPayable", err)
	}

	if len(repo.statusUpdates) != 1 || repo.statusUpdates[0].status != string(PaymentStatusPaid) {
		t.Fatalf("status updates = %+v, want satu PAID", repo.statusUpdates)
	}
}

func TestCreatePaymentAmbiguousStateChangesNothing(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)

	pending := newPendingPayment("PAY-UNSURE")
	repo := newFakePaymentRepo()
	repo.latest = pending
	repo.payments[pending.ID] = pending

	// Xendit tidak bisa dihubungi.
	xendit := &fakeXendit{foundErr: errors.New("connection refused")}
	svc := newTestService(&ord, repo, xendit)

	if _, err := svc.CreatePayment(context.Background(), userID, ord.ID); !errors.Is(err, ErrPaymentStateAmbiguous) {
		t.Fatalf("err = %v, want ErrPaymentStateAmbiguous", err)
	}

	if len(repo.statusUpdates) != 0 {
		t.Fatalf("status berubah = %+v, want tidak ada", repo.statusUpdates)
	}
	if len(repo.created) != 0 {
		t.Fatalf("attempt baru = %d, want 0", len(repo.created))
	}
}

func TestCreatePaymentCreateInvoiceConfirmedMissingMarksFailed(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)

	// Invoice terkonfirmasi tidak ada (nil, tanpa error).
	xendit := &fakeXendit{createErr: errors.New("500 boom")}
	repo := newFakePaymentRepo()

	svc := newTestService(&ord, repo, xendit)

	if _, err := svc.CreatePayment(context.Background(), userID, ord.ID); err == nil {
		t.Fatal("want error saat CreateInvoice gagal")
	}

	if len(repo.statusUpdates) != 1 || repo.statusUpdates[0].status != string(PaymentStatusFailed) {
		t.Fatalf("status updates = %+v, want satu FAILED", repo.statusUpdates)
	}
}

func TestCreatePaymentCreateInvoiceErrorUnverifiableStaysPending(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)

	// CreateInvoice gagal DAN verifikasi tak bisa dilakukan:
	// tidak boleh FAILED, row tetap PENDING.
	xendit := &fakeXendit{
		createErr: errors.New("timeout"),
		foundErr:  errors.New("connection refused"),
	}
	repo := newFakePaymentRepo()

	svc := newTestService(&ord, repo, xendit)

	if _, err := svc.CreatePayment(context.Background(), userID, ord.ID); !errors.Is(err, ErrPaymentStateAmbiguous) {
		t.Fatalf("err = %v, want ErrPaymentStateAmbiguous", err)
	}

	if len(repo.statusUpdates) != 0 {
		t.Fatalf("status berubah = %+v, want tidak ada", repo.statusUpdates)
	}
}

func TestCreatePaymentLosesRaceAndReusesWinner(t *testing.T) {
	userID := uuid.New()
	ord := payableOrder(userID)

	winner := newPendingPayment("PAY-WINNER")
	url := "https://checkout.example/winner"
	expiry := time.Now().Add(10 * time.Minute)
	winner.CheckoutURL = &url
	winner.ExpiresAt = &expiry

	repo := newFakePaymentRepo()
	repo.latest = winner
	// ponytail: hanya simulasi sekali kalah race; keduanya cukup untuk percaya jalannya.
	repo.createErr = &pgconn.PgError{Code: "23505", Message: "duplicate key"}

	xendit := &fakeXendit{}
	svc := newTestService(&ord, repo, xendit)

	result, err := svc.CreatePayment(context.Background(), userID, ord.ID)
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}

	if result.CheckoutURL != url {
		t.Fatalf("checkout_url = %q, want milik pemenang %q", result.CheckoutURL, url)
	}
	if xendit.createCalls != 0 {
		t.Fatalf("CreateInvoice dipanggil %d kali, want 0", xendit.createCalls)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestCreatePaymentRejectsFractionalAmount(t *testing.T) {
	ord := payableOrder(uuid.New())
	ord.TotalAmount = decimal.NewFromFloat(120000.5) // pecahan dari ongkir dsb.

	repo := newFakePaymentRepo()
	xendit := &fakeXendit{}
	svc := newTestService(&ord, repo, xendit)

	_, err := svc.CreatePayment(context.Background(), ord.UserID, ord.ID)
	if !errors.Is(err, ErrInvalidInvoiceAmount) {
		t.Fatalf("err = %v, want ErrInvalidInvoiceAmount", err)
	}
	if xendit.createCalls != 0 {
		t.Fatalf("CreateInvoice dipanggil %d kali, want 0", xendit.createCalls)
	}
}
