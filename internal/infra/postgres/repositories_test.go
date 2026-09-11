package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tomq29/ourmoney/internal/domain/category"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
	"github.com/tomq29/ourmoney/internal/domain/transaction"
)

func TestLedgerRoundTrip(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := NewLedgerRepo(testPool)

	created := seedLedger(t)

	found, err := repo.FindByID(ctx, created.ID())
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.Name() != "Home" || found.Currency().String() != "kzt" {
		t.Errorf("ledger = %q / %q", found.Name(), found.Currency().String())
	}
	if len(found.Members()) != 1 || !found.HasMember(1) {
		t.Fatalf("members = %+v", found.Members())
	}

	byMember, err := repo.FindByMember(ctx, 1)
	if err != nil {
		t.Fatalf("find by member: %v", err)
	}
	if byMember.ID() != created.ID() {
		t.Errorf("found the wrong ledger")
	}
}

func TestLedgerSavesMembershipChanges(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := NewLedgerRepo(testPool)

	l := seedLedger(t)
	partner, err := ledger.NewMember(2, "partner", ledger.RoleMember, time.Now())
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	if err := l.Join(partner); err != nil {
		t.Fatalf("join: %v", err)
	}

	if err := repo.Save(ctx, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	found, err := repo.FindByID(ctx, l.ID())
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(found.Members()) != 2 || !found.HasMember(2) {
		t.Fatalf("members = %+v", found.Members())
	}

	if err := l.Leave(2); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if err := repo.Save(ctx, l); err != nil {
		t.Fatalf("save after leaving: %v", err)
	}

	found, _ = repo.FindByID(ctx, l.ID())
	if found.HasMember(2) {
		t.Error("the member who left is still stored")
	}
}

func TestLedgerNotFound(t *testing.T) {
	reset(t)

	_, err := NewLedgerRepo(testPool).FindByMember(context.Background(), 999)
	if !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, common.ErrNotFound)
	}
}

// One Telegram account belongs to one ledger, so the bot never has to ask
// which budget a message is about.
func TestOnePersonCannotJoinTwoLedgers(t *testing.T) {
	reset(t)
	ctx := context.Background()

	seedLedger(t)

	owner, _ := ledger.NewMember(1, "tom", ledger.RoleOwner, time.Now())
	second, _ := ledger.NewLedger(uuid.NewString(), "Other", kzt(t), owner)

	if err := NewLedgerRepo(testPool).Create(ctx, second); err == nil {
		t.Fatal("the same person was allowed into a second ledger")
	}
}

func TestInviteIsClaimedOnce(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := NewInviteRepo(testPool)

	l := seedLedger(t)
	invite, err := ledger.NewInvite("ABC123", l.ID(), 1, time.Now(), time.Hour)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := repo.Create(ctx, invite); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.FindByCode(ctx, "ABC123")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.LedgerID() != l.ID() || found.IsUsed() {
		t.Fatalf("invite = %+v", found)
	}

	if err := found.Use(2, time.Now()); err != nil {
		t.Fatalf("use: %v", err)
	}
	if err := repo.Save(ctx, found); err != nil {
		t.Fatalf("save: %v", err)
	}

	// A second claim of the same code loses in the database, not in memory.
	stale, _ := repo.FindByCode(ctx, "ABC123")
	other := ledger.RestoreInvite(stale.Code(), stale.LedgerID(), stale.CreatedBy(),
		stale.CreatedAt(), stale.ExpiresAt(), 0, time.Time{})
	if err := other.Use(3, time.Now()); err != nil {
		t.Fatalf("use by the second person: %v", err)
	}
	if err := repo.Save(ctx, other); !errors.Is(err, ledger.ErrInviteUsed) {
		t.Fatalf("second save: %v, want %v", err, ledger.ErrInviteUsed)
	}
}

func TestCategoryRoundTrip(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := NewCategoryRepo(testPool)
	l := seedLedger(t)

	food, err := category.NewCategory(uuid.NewString(), l.ID(), "Еда", nil, time.Now())
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	if err := repo.Create(ctx, food); err != nil {
		t.Fatalf("create: %v", err)
	}

	groceries, _ := category.NewCategory(uuid.NewString(), l.ID(), "Продукты", food, time.Now())
	if err := repo.Create(ctx, groceries); err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Typing it in lower case must find the same category.
	found, err := repo.FindByName(ctx, l.ID(), "еда")
	if err != nil {
		t.Fatalf("find by name: %v", err)
	}
	if found.ID() != food.ID() {
		t.Errorf("found %q, want %q", found.Name(), food.Name())
	}

	all, err := repo.FindByLedgerID(ctx, l.ID())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("categories = %d, want 2", len(all))
	}
}

func TestCategoryNamesAreUniquePerLedger(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := NewCategoryRepo(testPool)
	l := seedLedger(t)

	first, _ := category.NewCategory(uuid.NewString(), l.ID(), "Еда", nil, time.Now())
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create: %v", err)
	}

	second, _ := category.NewCategory(uuid.NewString(), l.ID(), "еда", nil, time.Now())
	if err := repo.Create(ctx, second); err == nil {
		t.Fatal("the same category was created twice in different case")
	}
}

func TestTransactionRoundTripAndFilters(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := NewTransactionRepo(testPool)

	l := seedLedger(t)
	cash := seedAccount(t, l.ID(), "Cash", 100_000)
	card := seedAccount(t, l.ID(), "Card", 50_000)

	now := time.Now().UTC().Truncate(time.Millisecond)
	expense, err := transaction.NewExpense(transaction.NewExpenseParams{
		ID:            uuid.NewString(),
		LedgerID:      l.ID(),
		CreatedBy:     1,
		Amount:        money(t, 5_000),
		FromAccountID: cash.ID(),
		Comment:       "такси",
		Date:          now,
	})
	if err != nil {
		t.Fatalf("expense: %v", err)
	}
	if err := repo.Create(ctx, expense); err != nil {
		t.Fatalf("create expense: %v", err)
	}

	transfer, _ := transaction.NewTransfer(transaction.NewTransferParams{
		ID:            uuid.NewString(),
		LedgerID:      l.ID(),
		CreatedBy:     1,
		Amount:        money(t, 10_000),
		FromAccountID: cash.ID(),
		ToAccountID:   card.ID(),
		Date:          now.Add(time.Minute),
	})
	if err := repo.Create(ctx, transfer); err != nil {
		t.Fatalf("create transfer: %v", err)
	}

	stored, err := repo.FindByID(ctx, expense.ID())
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if stored.Comment() != "такси" || stored.CreatedBy() != 1 {
		t.Errorf("transaction = %+v", stored)
	}
	if !stored.Amount().IsEqual(money(t, 5_000)) {
		t.Errorf("amount = %s", stored.Amount().Amount())
	}
	if stored.Type() != transaction.Expense {
		t.Errorf("type = %v, want expense", stored.Type())
	}

	last, err := repo.Last(ctx, l.ID())
	if err != nil {
		t.Fatalf("last: %v", err)
	}
	if last.ID() != transfer.ID() {
		t.Error("the newest transaction is not the one returned")
	}

	expenses := transaction.Expense
	found, err := repo.Find(ctx, transaction.Filter{LedgerID: l.ID(), Type: &expenses})
	if err != nil {
		t.Fatalf("find by type: %v", err)
	}
	if len(found) != 1 || found[0].ID() != expense.ID() {
		t.Fatalf("filtered = %d rows", len(found))
	}

	byAccount, err := repo.Find(ctx, transaction.Filter{LedgerID: l.ID(), AccountID: card.ID()})
	if err != nil {
		t.Fatalf("find by account: %v", err)
	}
	if len(byAccount) != 1 || byAccount[0].ID() != transfer.ID() {
		t.Fatalf("by account = %d rows", len(byAccount))
	}

	if err := repo.Delete(ctx, expense.ID()); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Delete(ctx, expense.ID()); !errors.Is(err, common.ErrNotFound) {
		t.Errorf("deleting twice: %v, want %v", err, common.ErrNotFound)
	}
}

// The schema refuses a transaction that does not name the accounts its type
// requires, whatever the code above it believes.
func TestSchemaRejectsAnIncompleteTransfer(t *testing.T) {
	reset(t)
	ctx := context.Background()
	l := seedLedger(t)
	cash := seedAccount(t, l.ID(), "Cash", 100_000)

	_, err := testPool.Exec(ctx,
		`INSERT INTO transactions (id, ledger_id, created_by, transaction_type, amount, currency,
		                           from_account_id, to_account_id, date)
		 VALUES ($1, $2, 1, 'transfer', 100, 'kzt', $3, NULL, now())`,
		uuid.NewString(), l.ID(), cash.ID())

	if err == nil {
		t.Fatal("a transfer with one side was stored")
	}
}
