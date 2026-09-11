package ledger

import (
	"slices"
	"time"

	"github.com/tomq29/ourmoney/internal/domain/common"
)

// Ledger is the shared budget and the consistency boundary around membership:
// accounts, categories and transactions belong to it rather than to whoever
// typed them in. Two people spending from one wallet is the whole point.
type Ledger struct {
	id        string
	name      string
	currency  common.Currency
	members   []Member
	createdAt time.Time
}

func NewLedger(id, name string, currency common.Currency, owner Member) (*Ledger, error) {
	if len(name) == 0 {
		return nil, ErrEmptyName
	}
	if owner.telegramID <= 0 {
		return nil, ErrMissingTelegramID
	}

	owner.role = RoleOwner
	return &Ledger{
		id:        id,
		name:      name,
		currency:  currency,
		members:   []Member{owner},
		createdAt: owner.joinedAt,
	}, nil
}

func RestoreLedger(id, name string, currency common.Currency, members []Member, createdAt time.Time) *Ledger {
	return &Ledger{id: id, name: name, currency: currency, members: members, createdAt: createdAt}
}

func (l *Ledger) Join(m Member) error {
	if m.telegramID <= 0 {
		return ErrMissingTelegramID
	}
	if l.HasMember(m.telegramID) {
		return ErrAlreadyMember
	}

	l.members = append(l.members, m)
	return nil
}

func (l *Ledger) Leave(telegramID int64) error {
	i := slices.IndexFunc(l.members, func(m Member) bool { return m.telegramID == telegramID })
	if i < 0 {
		return ErrNotMember
	}

	// A ledger nobody owns cannot invite anyone, so it would be unreachable.
	if l.members[i].IsOwner() && l.owners() == 1 {
		return ErrLastOwner
	}

	l.members = slices.Delete(l.members, i, i+1)
	return nil
}

func (l *Ledger) Member(telegramID int64) (Member, bool) {
	i := slices.IndexFunc(l.members, func(m Member) bool { return m.telegramID == telegramID })
	if i < 0 {
		return Member{}, false
	}
	return l.members[i], true
}

func (l *Ledger) HasMember(telegramID int64) bool {
	_, ok := l.Member(telegramID)
	return ok
}

func (l *Ledger) owners() int {
	owners := 0
	for _, m := range l.members {
		if m.IsOwner() {
			owners++
		}
	}
	return owners
}

func (l *Ledger) ID() string                { return l.id }
func (l *Ledger) Name() string              { return l.name }
func (l *Ledger) Currency() common.Currency { return l.currency }
func (l *Ledger) Members() []Member         { return slices.Clone(l.members) }
func (l *Ledger) CreatedAt() time.Time      { return l.createdAt }
