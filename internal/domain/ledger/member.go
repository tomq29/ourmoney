package ledger

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

// Member is identified by the Telegram account writing to the bot. There is no
// separate identity to invent: Telegram already issued one, and it is stable.
type Member struct {
	telegramID int64
	name       string
	role       Role
	joinedAt   time.Time
}

func NewMember(telegramID int64, name string, role Role, joinedAt time.Time) (Member, error) {
	if telegramID <= 0 {
		return Member{}, ErrMissingTelegramID
	}
	if role != RoleOwner && role != RoleMember {
		return Member{}, ErrUnknownRole
	}
	return Member{telegramID: telegramID, name: name, role: role, joinedAt: joinedAt}, nil
}

func (m Member) TelegramID() int64   { return m.telegramID }
func (m Member) Name() string        { return m.name }
func (m Member) Role() Role          { return m.role }
func (m Member) JoinedAt() time.Time { return m.joinedAt }
func (m Member) IsOwner() bool       { return m.role == RoleOwner }
