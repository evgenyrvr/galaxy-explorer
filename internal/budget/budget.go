// Package budget implements the mass budget: non-overlapping matter
// categories that sum to 100% of M_total, and per-category shares between the
// mass-owning components. Every component mass is
//
//	M_c = M_total × f_category × share_c
//
// so each unit of mass has exactly one owner and structural labels never add mass.
package budget

import (
	"errors"
	"fmt"
	"math"
)

// Eps is the tolerance on sums of fractions (spec: ≤ 1e-6 relative).
const Eps = 1e-9

// Mode selects what stays fixed when one category is edited.
type Mode string

const (
	// FixTotal keeps M_total; fractions change and the remainder is handled by a Rule.
	FixTotal Mode = "fix_total"
	// FixComponents keeps the other categories' masses; M_total follows.
	FixComponents Mode = "fix_components"
)

// Rule decides what happens to the other categories in FixTotal mode.
type Rule string

const (
	// LeaveUnallocated changes one category only; the difference is shown as unallocated.
	LeaveUnallocated Rule = "leave_unallocated"
	// Proportional redistributes the difference over unlocked categories in proportion to their size.
	Proportional Rule = "proportional"
)

// Category is a top-level matter category.
type Category struct {
	ID       string  `json:"id"`
	Fraction float64 `json:"fraction"`
	Locked   bool    `json:"locked"`
}

// Share is a component's share of its category's mass.
type Share struct {
	Component string  `json:"component"`
	Share     float64 `json:"share"`
	Locked    bool    `json:"locked"`
}

// Budget is a complete, serialisable mass budget.
type Budget struct {
	TotalMass  float64            `json:"totalMass"`
	Mode       Mode               `json:"mode"`
	Categories []Category         `json:"categories"`
	Shares     map[string][]Share `json:"shares"`
}

// Errors returned by edit operations.
var (
	ErrNegative      = errors.New("fractions and masses must be non-negative")
	ErrOverAllocated = errors.New("the categories would exceed 100% of the total mass")
	ErrLocked        = errors.New("the value is locked; unlock it first")
	ErrNoRoom        = errors.New("not enough unlocked mass to redistribute")
	ErrUnknown       = errors.New("unknown category or component")
	ErrIncomplete    = errors.New("the budget must sum to 100% for this operation")
	ErrBadTotal      = errors.New("total mass must be a positive finite number")
)

// Clone returns a deep copy.
func (b Budget) Clone() Budget {
	c := b
	c.Categories = append([]Category(nil), b.Categories...)
	c.Shares = make(map[string][]Share, len(b.Shares))
	for k, v := range b.Shares {
		c.Shares[k] = append([]Share(nil), v...)
	}
	return c
}

func (b *Budget) cat(id string) (int, error) {
	for i, c := range b.Categories {
		if c.ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("%w: %q", ErrUnknown, id)
}

// Sum returns Σ category fractions.
func (b Budget) Sum() float64 {
	s := 0.0
	for _, c := range b.Categories {
		s += c.Fraction
	}
	return s
}

// Unallocated returns 1 − Σ fractions (positive = mass not yet assigned).
func (b Budget) Unallocated() float64 { return 1 - b.Sum() }

// Complete reports whether the categories sum to 100%.
func (b Budget) Complete() bool { return math.Abs(b.Unallocated()) <= Eps }

// CategoryFraction returns the fraction of a category (0 if absent).
func (b Budget) CategoryFraction(id string) float64 {
	for _, c := range b.Categories {
		if c.ID == id {
			return c.Fraction
		}
	}
	return 0
}

// CategoryMass returns M_total × f.
func (b Budget) CategoryMass(id string) float64 { return b.TotalMass * b.CategoryFraction(id) }

// ShareOf returns the share of component within category.
func (b Budget) ShareOf(cat, comp string) float64 {
	for _, s := range b.Shares[cat] {
		if s.Component == comp {
			return s.Share
		}
	}
	return 0
}

// ComponentMass returns M_total × f_cat × share.
func (b Budget) ComponentMass(cat, comp string) float64 {
	return b.TotalMass * b.CategoryFraction(cat) * b.ShareOf(cat, comp)
}

// Issue is a validation finding.
type Issue struct {
	Severity string `json:"severity"` // error | warning
	Message  string `json:"message"`
}

// Validate checks the invariants.
func (b Budget) Validate() []Issue {
	var out []Issue
	if !(b.TotalMass > 0) || math.IsInf(b.TotalMass, 0) {
		out = append(out, Issue{"error", "Total mass must be positive."})
	}
	for _, c := range b.Categories {
		if c.Fraction < 0 {
			out = append(out, Issue{"error", fmt.Sprintf("%s has a negative fraction.", c.ID)})
		}
	}
	u := b.Unallocated()
	switch {
	case u < -Eps:
		out = append(out, Issue{"error", fmt.Sprintf("Categories exceed 100%% by %.6g%%.", -u*100)})
	case u > Eps:
		out = append(out, Issue{"warning", fmt.Sprintf("%.6g%% of the total mass is unallocated.", u*100)})
	}
	for cat, ss := range b.Shares {
		sum := 0.0
		for _, s := range ss {
			if s.Share < 0 {
				out = append(out, Issue{"error", fmt.Sprintf("%s has a negative share.", s.Component)})
			}
			sum += s.Share
		}
		if len(ss) > 0 && math.Abs(sum-1) > 1e-6 {
			out = append(out, Issue{"error", fmt.Sprintf("Shares inside %s sum to %.6g%%, not 100%%.", cat, sum*100)})
		}
		if len(ss) == 0 && b.CategoryFraction(cat) > 0 {
			out = append(out, Issue{"error", fmt.Sprintf("%s has mass but no components to own it.", cat)})
		}
	}
	return out
}

// SetTotalMass changes M_total while keeping all fractions (AC-06).
func (b *Budget) SetTotalMass(m float64) error {
	if !(m > 0) || math.IsInf(m, 0) || math.IsNaN(m) {
		return ErrBadTotal
	}
	b.TotalMass = m
	return nil
}

// SetMode switches the editing mode. FixComponents requires a complete budget,
// because the "others" being held fixed must be well defined.
func (b *Budget) SetMode(m Mode) error {
	if m != FixTotal && m != FixComponents {
		return fmt.Errorf("unknown mode %q", m)
	}
	if m == FixComponents && !b.Complete() {
		return ErrIncomplete
	}
	b.Mode = m
	return nil
}

// SetLocked locks or unlocks a category.
func (b *Budget) SetLocked(id string, locked bool) error {
	i, err := b.cat(id)
	if err != nil {
		return err
	}
	b.Categories[i].Locked = locked
	return nil
}

// SetFraction sets a category's fraction of M_total.
//
// FixTotal: with LeaveUnallocated only this category changes and the sum may
// fall below 100% (shown as unallocated) but never exceed it; with
// Proportional the difference is taken from / given to unlocked categories.
//
// FixComponents: the other categories keep their masses, M_total and all
// fractions are recomputed.
func (b *Budget) SetFraction(id string, f float64, rule Rule) error {
	if f < 0 || math.IsNaN(f) {
		return ErrNegative
	}
	if f > 1 {
		return ErrOverAllocated
	}
	i, err := b.cat(id)
	if err != nil {
		return err
	}
	if b.Categories[i].Locked {
		return ErrLocked
	}
	if b.Mode == FixComponents {
		if f >= 1 {
			return ErrOverAllocated
		}
		rest := b.TotalMass * (1 - b.Categories[i].Fraction)
		if rest <= 0 {
			return ErrNoRoom
		}
		return b.SetCategoryMass(id, f*rest/(1-f))
	}
	switch rule {
	case LeaveUnallocated, "":
		if b.Sum()-b.Categories[i].Fraction+f > 1+Eps {
			return ErrOverAllocated
		}
		b.Categories[i].Fraction = f
		return nil
	case Proportional:
		delta := f - b.Categories[i].Fraction
		return b.redistribute(i, f, delta)
	default:
		return fmt.Errorf("unknown rule %q", rule)
	}
}

// redistribute sets category i to f and absorbs delta in unlocked others.
func (b *Budget) redistribute(i int, f, delta float64) error {
	avail := 0.0
	var idx []int
	for j, c := range b.Categories {
		if j != i && !c.Locked {
			idx = append(idx, j)
			avail += c.Fraction
		}
	}
	if len(idx) == 0 {
		return ErrNoRoom
	}
	if delta > avail+Eps {
		return ErrNoRoom
	}
	b.Categories[i].Fraction = f
	if avail > 0 {
		k := (avail - delta) / avail
		for _, j := range idx {
			b.Categories[j].Fraction *= k
		}
	} else {
		// All unlocked others are empty: share the freed mass equally.
		for _, j := range idx {
			b.Categories[j].Fraction = -delta / float64(len(idx))
		}
	}
	for _, j := range idx {
		if b.Categories[j].Fraction < 0 {
			b.Categories[j].Fraction = 0
		}
	}
	return nil
}

// SetCategoryMass sets a category's absolute mass (M☉).
func (b *Budget) SetCategoryMass(id string, m float64) error {
	if m < 0 || math.IsNaN(m) || math.IsInf(m, 0) {
		return ErrNegative
	}
	i, err := b.cat(id)
	if err != nil {
		return err
	}
	if b.Categories[i].Locked {
		return ErrLocked
	}
	if b.Mode != FixComponents {
		return b.SetFraction(id, m/b.TotalMass, LeaveUnallocated)
	}
	masses := make([]float64, len(b.Categories))
	total := 0.0
	for j, c := range b.Categories {
		masses[j] = c.Fraction * b.TotalMass
		if j == i {
			masses[j] = m
		}
		total += masses[j]
	}
	if !(total > 0) {
		return ErrBadTotal
	}
	b.TotalMass = total
	for j := range b.Categories {
		b.Categories[j].Fraction = masses[j] / total
	}
	return nil
}

// Normalize scales the unlocked categories so the budget sums to 100%.
func (b *Budget) Normalize() error {
	locked, unlocked := 0.0, 0.0
	for _, c := range b.Categories {
		if c.Locked {
			locked += c.Fraction
		} else {
			unlocked += c.Fraction
		}
	}
	if locked > 1+Eps {
		return ErrOverAllocated
	}
	if unlocked <= 0 {
		return ErrNoRoom
	}
	k := (1 - locked) / unlocked
	for i := range b.Categories {
		if !b.Categories[i].Locked {
			b.Categories[i].Fraction *= k
		}
	}
	return nil
}

// AssignRemainder gives the unallocated fraction (positive or negative) to one category.
func (b *Budget) AssignRemainder(id string) error {
	i, err := b.cat(id)
	if err != nil {
		return err
	}
	if b.Categories[i].Locked {
		return ErrLocked
	}
	f := b.Categories[i].Fraction + b.Unallocated()
	if f < -Eps {
		return ErrNegative
	}
	b.Categories[i].Fraction = math.Max(0, f)
	return nil
}

// SetShare sets a component's share inside its category; the other unlocked
// shares of the category are rescaled so the shares keep summing to 100%.
func (b *Budget) SetShare(cat, comp string, s float64) error {
	if s < 0 || math.IsNaN(s) {
		return ErrNegative
	}
	if s > 1 {
		return ErrOverAllocated
	}
	ss, ok := b.Shares[cat]
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknown, cat)
	}
	i := -1
	for j := range ss {
		if ss[j].Component == comp {
			i = j
		}
	}
	if i < 0 {
		return fmt.Errorf("%w: %q", ErrUnknown, comp)
	}
	if ss[i].Locked {
		return ErrLocked
	}
	locked, avail := 0.0, 0.0
	var idx []int
	for j, x := range ss {
		switch {
		case j == i:
		case x.Locked:
			locked += x.Share
		default:
			idx = append(idx, j)
			avail += x.Share
		}
	}
	if s+locked > 1+Eps {
		return ErrNoRoom
	}
	if len(idx) == 0 {
		if math.Abs(s+locked-1) > Eps {
			return ErrNoRoom
		}
		ss[i].Share = s
		return nil
	}
	remaining := 1 - locked - s
	ss[i].Share = s
	if avail > 0 {
		k := remaining / avail
		for _, j := range idx {
			ss[j].Share *= k
		}
	} else {
		for _, j := range idx {
			ss[j].Share = remaining / float64(len(idx))
		}
	}
	return nil
}

// SetShareLocked locks or unlocks a component share.
func (b *Budget) SetShareLocked(cat, comp string, locked bool) error {
	for j := range b.Shares[cat] {
		if b.Shares[cat][j].Component == comp {
			b.Shares[cat][j].Locked = locked
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrUnknown, comp)
}

// History is an undo/redo stack of budget snapshots.
type History struct {
	past, future []Budget
	limit        int
}

// NewHistory returns a history with the given depth.
func NewHistory(limit int) *History { return &History{limit: limit} }

// Push records the state before a change and clears the redo stack.
func (h *History) Push(before Budget) {
	h.past = append(h.past, before.Clone())
	if len(h.past) > h.limit {
		h.past = h.past[1:]
	}
	h.future = nil
}

// Undo returns the previous state.
func (h *History) Undo(cur Budget) (Budget, bool) {
	if len(h.past) == 0 {
		return cur, false
	}
	prev := h.past[len(h.past)-1]
	h.past = h.past[:len(h.past)-1]
	h.future = append(h.future, cur.Clone())
	return prev, true
}

// Redo re-applies an undone state.
func (h *History) Redo(cur Budget) (Budget, bool) {
	if len(h.future) == 0 {
		return cur, false
	}
	next := h.future[len(h.future)-1]
	h.future = h.future[:len(h.future)-1]
	h.past = append(h.past, cur.Clone())
	return next, true
}

// CanUndo / CanRedo report stack availability.
func (h *History) CanUndo() bool { return len(h.past) > 0 }
func (h *History) CanRedo() bool { return len(h.future) > 0 }

// Clear empties both stacks.
func (h *History) Clear() { h.past, h.future = nil, nil }
