package ledger

import "time"

// Invite is how the second person gets in. It replaces registration entirely:
// there is no password to choose, only a code that expires and works once.
type Invite struct {
	code      string
	ledgerID  string
	createdBy int64
	createdAt time.Time
	expiresAt time.Time
	usedBy    int64
	usedAt    time.Time
}

func NewInvite(code, ledgerID string, createdBy int64, now time.Time, ttl time.Duration) (*Invite, error) {
	if len(code) == 0 {
		return nil, ErrEmptyCode
	}
	if createdBy <= 0 {
		return nil, ErrMissingTelegramID
	}

	return &Invite{
		code:      code,
		ledgerID:  ledgerID,
		createdBy: createdBy,
		createdAt: now,
		expiresAt: now.Add(ttl),
	}, nil
}

func RestoreInvite(code, ledgerID string, createdBy int64, createdAt, expiresAt time.Time, usedBy int64, usedAt time.Time) *Invite {
	return &Invite{
		code:      code,
		ledgerID:  ledgerID,
		createdBy: createdBy,
		createdAt: createdAt,
		expiresAt: expiresAt,
		usedBy:    usedBy,
		usedAt:    usedAt,
	}
}

func (i *Invite) Use(telegramID int64, now time.Time) error {
	if telegramID <= 0 {
		return ErrMissingTelegramID
	}
	if i.IsUsed() {
		return ErrInviteUsed
	}
	if now.After(i.expiresAt) {
		return ErrInviteExpired
	}

	i.usedBy, i.usedAt = telegramID, now
	return nil
}

func (i *Invite) IsUsed() bool         { return i.usedBy != 0 }
func (i *Invite) Code() string         { return i.code }
func (i *Invite) LedgerID() string     { return i.ledgerID }
func (i *Invite) CreatedBy() int64     { return i.createdBy }
func (i *Invite) CreatedAt() time.Time { return i.createdAt }
func (i *Invite) ExpiresAt() time.Time { return i.expiresAt }
func (i *Invite) UsedBy() int64        { return i.usedBy }
func (i *Invite) UsedAt() time.Time    { return i.usedAt }
