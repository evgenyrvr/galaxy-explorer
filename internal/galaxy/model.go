package galaxy

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"sync"

	"galaxy/internal/budget"
	"galaxy/internal/ontology"
	"galaxy/internal/physics"
	"galaxy/internal/units"
)

// Model is a live galaxy: preset + budget + components. It is safe for
// concurrent use.
type Model struct {
	mu        sync.Mutex
	preset    Preset
	budget    budget.Budget
	hist      *budget.History
	cosmo     units.Cosmology
	seed      int64
	comps     []*Component
	byID      map[string]*Component
	tags      []ontology.Tag
	tagBit    map[string]uint
	r200c     float64
	partCache map[string][]byte
	refCache  map[string]refSample
}

// NewModel builds a model from a preset with its default budget.
func NewModel(p Preset, seed int64) (*Model, error) {
	m := &Model{preset: p, cosmo: units.Planck18, seed: seed, hist: budget.NewHistory(200), partCache: map[string][]byte{}}
	if err := m.setBudget(p.DefaultBudget()); err != nil {
		return nil, err
	}
	return m, nil
}

// setBudget installs b and rebuilds geometry if R200c changed.
func (m *Model) setBudget(b budget.Budget) error {
	r200 := m.cosmo.R200c(b.TotalMass)
	if m.comps == nil || math.Abs(r200-m.r200c) > 1e-9*r200 {
		comps := m.preset.Build(Env{TotalMass: b.TotalMass, R200c: r200, Cosmology: m.cosmo, Seed: m.seed})
		if err := checkOwnership(b, comps); err != nil {
			return err
		}
		m.comps = comps
		m.byID = map[string]*Component{}
		for _, c := range comps {
			m.byID[c.ID] = c
		}
		m.r200c = r200
		m.indexTags()
	}
	m.budget = b
	return nil
}

// checkOwnership verifies that every component owns exactly one share and
// every share belongs to a component of the right matter type.
func checkOwnership(b budget.Budget, comps []*Component) error {
	seen := map[string]bool{}
	for cat, ss := range b.Shares {
		for _, s := range ss {
			if seen[s.Component] {
				return fmt.Errorf("component %q owns mass in two categories", s.Component)
			}
			seen[s.Component] = true
			found := false
			for _, c := range comps {
				if c.ID == s.Component {
					found = true
					if string(c.Matter) != cat {
						return fmt.Errorf("component %q is %s but listed under %s", c.ID, c.Matter, cat)
					}
				}
			}
			if !found {
				return fmt.Errorf("share for unknown component %q", s.Component)
			}
		}
	}
	for _, c := range comps {
		if !seen[c.ID] {
			return fmt.Errorf("component %q has no mass share", c.ID)
		}
	}
	return nil
}

func (m *Model) indexTags() {
	m.tags = m.preset.Tags()
	m.tagBit = map[string]uint{}
	for i, t := range m.tags {
		m.tagBit[t.ID] = uint(i)
	}
	// Any tag used by components but missing from the preset registry.
	for _, c := range m.comps {
		names := append(append([]string{}, c.Tags...), c.Dist.TagNames()...)
		for _, n := range names {
			if _, ok := m.tagBit[n]; !ok {
				m.tagBit[n] = uint(len(m.tags))
				m.tags = append(m.tags, ontology.Tag{ID: n, Label: n, Family: "other"})
			}
		}
	}
	if len(m.tags) > 24 {
		panic("galaxy: more than 24 structural tags (GPU bitmask limit)")
	}
}

// Preset returns the preset.
func (m *Model) Preset() Preset { return m.preset }

// Budget returns a copy of the current budget.
func (m *Model) Budget() budget.Budget {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.budget.Clone()
}

// Seed returns the generation seed.
func (m *Model) Seed() int64 { return m.seed }

// Edit applies fn to a copy of the budget; on success the previous budget is
// pushed to the undo history. On error nothing changes.
func (m *Model) Edit(fn func(b *budget.Budget) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	nb := m.budget.Clone()
	if err := fn(&nb); err != nil {
		return err
	}
	for _, is := range nb.Validate() {
		if is.Severity == "error" && !isAllocationIssue(is) {
			return fmt.Errorf("%s", is.Message)
		}
	}
	prev := m.budget.Clone()
	if err := m.setBudget(nb); err != nil {
		return err
	}
	m.hist.Push(prev)
	return nil
}

// isAllocationIssue reports issues that are shown to the user rather than
// blocking (an incomplete budget is a legal intermediate state).
func isAllocationIssue(is budget.Issue) bool {
	return is.Severity == "warning"
}

// Undo / Redo / Reset.
func (m *Model) Undo() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.hist.Undo(m.budget)
	if ok {
		_ = m.setBudget(b)
	}
	return ok
}

func (m *Model) Redo() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.hist.Redo(m.budget)
	if ok {
		_ = m.setBudget(b)
	}
	return ok
}

// Reset restores the preset's default budget (undoable).
func (m *Model) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	prev := m.budget.Clone()
	_ = m.setBudget(m.preset.DefaultBudget())
	m.hist.Push(prev)
}

// Replace installs a budget loaded from a project file.
func (m *Model) Replace(b budget.Budget, seed int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, is := range b.Validate() {
		if is.Severity == "error" {
			return fmt.Errorf("invalid budget in project: %s", is.Message)
		}
	}
	if seed != m.seed {
		m.seed = seed
		m.comps = nil
		m.partCache = map[string][]byte{}
		m.refCache = nil
	}
	if err := m.setBudget(b); err != nil {
		return err
	}
	m.hist.Clear()
	return nil
}

// ComponentMass returns the physical mass of a component (M☉).
func (m *Model) componentMass(c *Component) float64 {
	return m.budget.ComponentMass(string(c.Matter), c.ID)
}

// ---------------------------------------------------------------------------
// State snapshot for the UI.

// CategoryState is one budget row.
type CategoryState struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Group    string  `json:"group"`
	Baryonic bool    `json:"baryonic"`
	Note     string  `json:"note"`
	Fraction float64 `json:"fraction"`
	Mass     float64 `json:"mass"`
	Locked   bool    `json:"locked"`
}

// ComponentState is one mass owner.
type ComponentState struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Matter          string             `json:"matter"`
	Structure       string             `json:"structure"`
	Tags            []string           `json:"tags"`
	LocalTags       []string           `json:"localTags"`
	Evidence        string             `json:"evidence"`
	Sources         []string           `json:"sources"`
	Notes           string             `json:"notes"`
	Kind            string             `json:"kind"`
	Params          map[string]float64 `json:"params"`
	Mass            float64            `json:"mass"`
	Share           float64            `json:"share"`
	ShareLocked     bool               `json:"shareLocked"`
	FractionOfTotal float64            `json:"fractionOfTotal"`
	InsideR200c     float64            `json:"insideR200c"` // fraction of the component's mass inside R200c
	MaxRadius       float64            `json:"maxRadius"`
	Visual          Visual             `json:"visual"`
	Markers         int                `json:"markers"`
	FixedCount      int                `json:"fixedCount"` // >0: discrete physical objects
	GeomKey         string             `json:"geomKey"`
}

// TagOwner is one component's contribution to a structural tag.
type TagOwner struct {
	Component string  `json:"component"`
	Fraction  float64 `json:"fraction"` // of the component's mass
	Mass      float64 `json:"mass"`
}

// TagState is a structural tag with its (non-additive) mass.
type TagState struct {
	ontology.Tag
	Bit    uint       `json:"bit"`
	Mass   float64    `json:"mass"`
	Exact  bool       `json:"exact"`
	Owners []TagOwner `json:"owners"`
}

// State is the full snapshot sent to the UI after every change.
type State struct {
	Preset               PresetInfo       `json:"preset"`
	Cosmology            units.Cosmology  `json:"cosmology"`
	HaloDefinition       string           `json:"haloDefinition"`
	TotalMass            float64          `json:"totalMass"`
	R200c                float64          `json:"r200c"`
	V200c                float64          `json:"v200c"`
	Concentration        float64          `json:"concentration"`
	Mode                 string           `json:"mode"`
	Categories           []CategoryState  `json:"categories"`
	Unallocated          float64          `json:"unallocated"`
	UnallocatedMass      float64          `json:"unallocatedMass"`
	Issues               []budget.Issue   `json:"issues"`
	Components           []ComponentState `json:"components"`
	Tags                 []TagState       `json:"tags"`
	BaryonFraction       float64          `json:"baryonFraction"`
	CosmicBaryonFraction float64          `json:"cosmicBaryonFraction"`
	Modified             bool             `json:"modified"` // budget differs from the preset defaults
	CanUndo              bool             `json:"canUndo"`
	CanRedo              bool             `json:"canRedo"`
	Seed                 int64            `json:"seed"`
	Sources              []Source         `json:"sources"`
}

// Snapshot returns the current state.
func (m *Model) Snapshot() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.budget
	s := State{
		Preset:               m.preset.Info(),
		Cosmology:            m.cosmo,
		HaloDefinition:       "M200c: all matter inside R200c, where the mean density is 200 ρcrit. Satellites inside R200c are already part of M200c.",
		TotalMass:            b.TotalMass,
		R200c:                m.r200c,
		V200c:                m.cosmo.V200c(b.TotalMass),
		Concentration:        m.cosmo.ConcentrationDM14(b.TotalMass),
		Mode:                 string(b.Mode),
		Unallocated:          b.Unallocated(),
		UnallocatedMass:      b.Unallocated() * b.TotalMass,
		Issues:               b.Validate(),
		Modified:             !sameBudget(b, m.preset.DefaultBudget()),
		CanUndo:              m.hist.CanUndo(),
		CanRedo:              m.hist.CanRedo(),
		CosmicBaryonFraction: m.cosmo.BaryonFraction(),
		Seed:                 m.seed,
	}
	// The concentration of the preset's own NFW halo, when it has one.
	for _, c := range m.comps {
		if c.Matter == ontology.DarkMatter && c.Dist.Kind() == "nfw" {
			if cc, ok := c.Dist.Params()["c200c"]; ok {
				s.Concentration = cc
			}
		}
	}
	for _, c := range b.Categories {
		info, _ := ontology.MatterByID(ontology.MatterType(c.ID))
		s.Categories = append(s.Categories, CategoryState{
			ID: c.ID, Label: info.Label, Group: info.Group, Baryonic: info.Baryonic, Note: info.Note,
			Fraction: c.Fraction, Mass: c.Fraction * b.TotalMass, Locked: c.Locked,
		})
		if info.Baryonic {
			s.BaryonFraction += c.Fraction
		}
	}
	used := map[string]bool{}
	for _, c := range m.comps {
		mass := m.componentMass(c)
		fc := 0
		if f, ok := c.Dist.(physics.FixedCounter); ok {
			fc = f.FixedCount()
		}
		locked := false
		for _, sh := range b.Shares[string(c.Matter)] {
			if sh.Component == c.ID {
				locked = sh.Locked
			}
		}
		s.Components = append(s.Components, ComponentState{
			ID: c.ID, Name: c.Name, Matter: string(c.Matter), Structure: c.Structure,
			Tags: c.Tags, LocalTags: c.Dist.TagNames(), Evidence: string(c.Evidence), Sources: c.Sources,
			Notes: c.Notes, Kind: c.Dist.Kind(), Params: c.Dist.Params(), Mass: mass,
			Share: b.ShareOf(string(c.Matter), c.ID), ShareLocked: locked,
			FractionOfTotal: mass / b.TotalMass, InsideR200c: c.Dist.SphericalFraction(m.r200c),
			MaxRadius: c.Dist.MaxRadius(), Visual: c.Visual, Markers: c.Visual.Markers, FixedCount: fc, GeomKey: c.GeomKey,
		})
		for _, id := range c.Sources {
			used[id] = true
		}
	}
	for _, t := range m.tags {
		ts := TagState{Tag: t, Bit: m.tagBit[t.ID], Exact: true}
		for _, c := range m.comps {
			f, exact := m.tagFraction(c, t.ID)
			if f > 0 || m.componentCarries(c, t.ID) {
				ts.Owners = append(ts.Owners, TagOwner{Component: c.ID, Fraction: f, Mass: f * m.componentMass(c)})
				ts.Mass += f * m.componentMass(c)
				ts.Exact = ts.Exact && exact
			}
		}
		s.Tags = append(s.Tags, ts)
	}
	ids := make([]string, 0, len(used))
	for id := range used {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if src, ok := Sources[id]; ok {
			s.Sources = append(s.Sources, src)
		}
	}
	return s
}

func (m *Model) componentCarries(c *Component, tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	for _, t := range c.Dist.TagNames() {
		if t == tag {
			return true
		}
	}
	return false
}

// tagFraction returns the fraction of c's mass carrying tag.
func (m *Model) tagFraction(c *Component, tag string) (float64, bool) {
	for _, t := range c.Tags {
		if t == tag {
			return 1, true
		}
	}
	for k, t := range c.Dist.TagNames() {
		if t == tag {
			return c.Dist.TagFraction(k + 1), c.Dist.TagFractionExact()
		}
	}
	return 0, true
}

// ---------------------------------------------------------------------------
// Selection.

// Selection is a union of whole components and structural tags.
type Selection struct {
	Components []string `json:"components"`
	Tags       []string `json:"tags"`
	// Region, if active, intersects the selection with a volume. With an
	// empty component/tag list the region alone selects all components.
	Region *Region `json:"region,omitempty"`
}

// SelectionPart is one component's contribution to a selection.
type SelectionPart struct {
	Component string  `json:"component"`
	Matter    string  `json:"matter"`
	Fraction  float64 `json:"fraction"` // of the component's mass
	Mass      float64 `json:"mass"`
}

// SelectionReport is the mass of a selection, counted once per mass owner.
type SelectionReport struct {
	Mass            float64            `json:"mass"`
	FractionOfTotal float64            `json:"fractionOfTotal"`
	ByMatter        map[string]float64 `json:"byMatter"`
	Parts           []SelectionPart    `json:"parts"`
	Exact           bool               `json:"exact"`
	InsideR200c     float64            `json:"insideR200c"` // mass of the selection inside R200c
}

// SelectionMass computes the mass of a selection. Each component contributes
// at most its own mass: a component selected as a whole is not added again
// through its tags, and disjoint tags of one component are summed.
func (m *Model) SelectionMass(sel Selection) SelectionReport {
	m.mu.Lock()
	defer m.mu.Unlock()
	whole := map[string]bool{}
	for _, id := range sel.Components {
		whole[id] = true
	}
	region := sel.Region
	if !region.Active() {
		region = nil
	}
	everything := region != nil && len(sel.Components) == 0 && len(sel.Tags) == 0
	r := SelectionReport{ByMatter: map[string]float64{}, Exact: true}
	for _, c := range m.comps {
		isWhole := whole[c.ID] || everything
		local := map[int]bool{}
		f := 0.0
		if isWhole {
			f = 1
		} else {
			for _, t := range sel.Tags {
				tf, exact := m.tagFraction(c, t)
				if tf == 1 {
					isWhole = true
					f = 1
					break
				}
				if tf > 0 {
					f += tf
					r.Exact = r.Exact && exact
					for k, name := range c.Dist.TagNames() {
						if name == t {
							local[k+1] = true
						}
					}
				}
			}
			f = math.Min(f, 1)
		}
		if f == 0 {
			continue
		}
		if region != nil {
			rf, exact := m.regionFraction(c, isWhole, local, region)
			f = rf
			r.Exact = r.Exact && exact
			if f == 0 {
				continue
			}
		}
		mass := f * m.componentMass(c)
		r.Parts = append(r.Parts, SelectionPart{Component: c.ID, Matter: string(c.Matter), Fraction: f, Mass: mass})
		r.Mass += mass
		r.ByMatter[string(c.Matter)] += mass
		inside := c.Dist.SphericalFraction(m.r200c)
		if region != nil && region.Kind == "sphere" && region.RMax <= m.r200c {
			inside = 1
		}
		r.InsideR200c += mass * math.Min(1, inside)
	}
	r.FractionOfTotal = r.Mass / m.budget.TotalMass
	return r
}

// ---------------------------------------------------------------------------
// Visual markers.

// Particles returns the binary marker buffer of a component:
//
//	uint32 n | float32 xyz × n | uint32 tag bitmask × n   (little endian)
//
// Markers are representative: each carries weight M/n and is not a star.
func (m *Model) Particles(id string, n int) ([]byte, error) {
	m.mu.Lock()
	c, ok := m.byID[id]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("unknown component %q", id)
	}
	if f, ok := c.Dist.(physics.FixedCounter); ok && f.FixedCount() > 0 {
		n = f.FixedCount()
	}
	if n < 1 {
		n = 1
	}
	key := fmt.Sprintf("%s|%s|%d|%d", id, c.GeomKey, n, m.seed)
	if buf, ok := m.partCache[key]; ok {
		m.mu.Unlock()
		return buf, nil
	}
	dist := c.Dist
	wholeMask := uint32(0)
	for _, t := range c.Tags {
		wholeMask |= 1 << m.tagBit[t]
	}
	local := dist.TagNames()
	localBits := make([]uint32, len(local)+1)
	for k, t := range local {
		localBits[k+1] = 1 << m.tagBit[t]
	}
	seed := m.seed ^ int64(hash64(key))
	m.mu.Unlock()

	rng := rand.New(rand.NewSource(seed))
	pos := make([]float32, 3*n)
	tags := make([]uint8, n)
	dist.Sample(rng, pos, tags)
	buf := make([]byte, 4+16*n)
	binary.LittleEndian.PutUint32(buf, uint32(n))
	off := 4
	for _, v := range pos {
		binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(v))
		off += 4
	}
	for _, t := range tags {
		binary.LittleEndian.PutUint32(buf[off:], wholeMask|localBits[t])
		off += 4
	}
	m.mu.Lock()
	if len(m.partCache) > 256 {
		m.partCache = map[string][]byte{}
	}
	m.partCache[key] = buf
	m.mu.Unlock()
	return buf, nil
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// sameBudget compares masses and shares within the budget tolerance.
func sameBudget(a, b budget.Budget) bool {
	if math.Abs(a.TotalMass/b.TotalMass-1) > 1e-9 || len(a.Categories) != len(b.Categories) {
		return false
	}
	for _, c := range b.Categories {
		if math.Abs(a.CategoryFraction(c.ID)-c.Fraction) > 1e-12 {
			return false
		}
		for _, sh := range b.Shares[c.ID] {
			if math.Abs(a.ShareOf(c.ID, sh.Component)-sh.Share) > 1e-12 {
				return false
			}
		}
	}
	return true
}
