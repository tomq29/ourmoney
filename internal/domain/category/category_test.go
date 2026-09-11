package category

import (
	"errors"
	"testing"
	"time"
)

func TestNewCategoryIsRootWithoutAParent(t *testing.T) {
	c, err := NewCategory("id", "ledger", "Food", nil, time.Now())
	if err != nil {
		t.Fatalf("category: %v", err)
	}

	if !c.IsRoot() || c.ParentID() != "" {
		t.Errorf("parent = %q, want none", c.ParentID())
	}
}

func TestNewCategoryUnderAParent(t *testing.T) {
	parent, err := NewCategory("parent", "ledger", "Food", nil, time.Now())
	if err != nil {
		t.Fatalf("parent: %v", err)
	}

	child, err := NewCategory("child", "ledger", "Groceries", parent, time.Now())
	if err != nil {
		t.Fatalf("child: %v", err)
	}
	if child.ParentID() != "parent" {
		t.Errorf("parent = %q, want parent", child.ParentID())
	}
	if child.IsRoot() {
		t.Error("a child reports itself as a root")
	}
}

func TestCategoriesNestOnlyOneLevel(t *testing.T) {
	parent, _ := NewCategory("parent", "ledger", "Food", nil, time.Now())
	child, _ := NewCategory("child", "ledger", "Groceries", parent, time.Now())

	if _, err := NewCategory("grandchild", "ledger", "Milk", child, time.Now()); !errors.Is(err, ErrTooDeep) {
		t.Fatalf("err = %v, want %v", err, ErrTooDeep)
	}
}

func TestCategoryCannotCrossLedgers(t *testing.T) {
	parent, _ := NewCategory("parent", "theirs", "Food", nil, time.Now())

	if _, err := NewCategory("child", "ours", "Groceries", parent, time.Now()); !errors.Is(err, ErrForeignParent) {
		t.Fatalf("err = %v, want %v", err, ErrForeignParent)
	}
}

func TestNewCategoryRejectsEmptyName(t *testing.T) {
	if _, err := NewCategory("id", "ledger", "", nil, time.Now()); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("err = %v, want %v", err, ErrEmptyName)
	}
}
