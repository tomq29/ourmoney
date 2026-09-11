package category

import (
	"errors"
	"time"
)

var (
	ErrEmptyName     = errors.New("category name is empty")
	ErrTooDeep       = errors.New("categories nest only one level")
	ErrForeignParent = errors.New("parent category belongs to another ledger")
)

// Category nests one level: "Food" holds "Groceries" and "Eating out", and
// that is as far as it goes. Deeper trees make reports harder to read without
// making them more useful, and they are what makes re-parenting able to build
// a cycle. Categories cannot be re-parented here, so no cycle can form.
type Category struct {
	id        string
	ledgerID  string
	name      string
	parentID  string
	createdAt time.Time
}

func NewCategory(id, ledgerID, name string, parent *Category, createdAt time.Time) (*Category, error) {
	if len(name) == 0 {
		return nil, ErrEmptyName
	}

	parentID := ""
	if parent != nil {
		if parent.ledgerID != ledgerID {
			return nil, ErrForeignParent
		}
		if parent.parentID != "" {
			return nil, ErrTooDeep
		}
		parentID = parent.id
	}

	return &Category{
		id:        id,
		ledgerID:  ledgerID,
		name:      name,
		parentID:  parentID,
		createdAt: createdAt,
	}, nil
}

func RestoreCategory(id, ledgerID, name, parentID string, createdAt time.Time) *Category {
	return &Category{id: id, ledgerID: ledgerID, name: name, parentID: parentID, createdAt: createdAt}
}

func (c *Category) ID() string           { return c.id }
func (c *Category) LedgerID() string     { return c.ledgerID }
func (c *Category) Name() string         { return c.name }
func (c *Category) ParentID() string     { return c.parentID }
func (c *Category) IsRoot() bool         { return c.parentID == "" }
func (c *Category) CreatedAt() time.Time { return c.createdAt }
