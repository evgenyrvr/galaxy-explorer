// Package ontology defines the two independent classifications of every
// galaxy element — matter type (mass owner category) and structure — and the
// scientific evidence levels.
package ontology

// MatterType is a non-overlapping physical mass category. The categories of
// one galaxy always sum to 100% of M_total.
type MatterType string

const (
	Stars          MatterType = "stars"
	GasHI          MatterType = "gas.hi"
	GasH2          MatterType = "gas.h2"
	GasWarmIonized MatterType = "gas.warm_ionized"
	GasHotIonized  MatterType = "gas.hot_ionized"
	Dust           MatterType = "dust"
	DarkMatter     MatterType = "dark_matter"
	BlackHole      MatterType = "black_hole"
)

// MatterInfo describes a matter category.
type MatterInfo struct {
	ID       MatterType `json:"id"`
	Label    string     `json:"label"`
	Group    string     `json:"group"`    // tree grouping: Stars, Gas, Dust, Dark matter, Black hole
	Baryonic bool       `json:"baryonic"` // counts toward the baryon fraction
	Note     string     `json:"note"`
}

// Matter lists all categories in display order.
var Matter = []MatterInfo{
	{DarkMatter, "Dark matter", "Dark matter", false, "Gravitationally inferred; drawn as a conventional translucent cloud, it does not emit light."},
	{Stars, "Stars", "Stars", true, "All stellar structures; each star is counted once regardless of structural labels."},
	{GasHI, "Atomic gas H I", "Gas", true, ""},
	{GasH2, "Molecular gas H₂", "Gas", true, "Usually inferred from CO with a conversion factor."},
	{GasWarmIonized, "Warm ionized gas", "Gas", true, "~10⁴ K: diffuse ionized gas and H II regions."},
	{GasHotIonized, "Hot ionized gas", "Gas", true, "≳10⁶ K corona / circumgalactic medium."},
	{Dust, "Dust", "Dust", true, "Counted as its own category, never inside the gas budget."},
	{BlackHole, "Central black hole", "Black hole", false, "Part of the gravitating mass; not counted as stars or gas. Drawn as a schematic icon."},
}

// MatterByID returns the info for a matter type.
func MatterByID(id MatterType) (MatterInfo, bool) {
	for _, m := range Matter {
		if m.ID == id {
			return m, true
		}
	}
	return MatterInfo{}, false
}

// Evidence is the scientific provenance level of an entity or parameter.
type Evidence string

const (
	// Observed: directly measured (positions on the sky, fluxes, spectra, v_los).
	Observed Evidence = "OBSERVED"
	// Inferred: derived from observations under assumptions (M* from light, H₂ from CO, DM).
	Inferred Evidence = "INFERRED"
	// Reconstructed: 3D placement/profile from fits or models.
	Reconstructed Evidence = "RECONSTRUCTED"
	// Illustrative: educational element with no claim of a specific observed structure.
	Illustrative Evidence = "ILLUSTRATIVE"
)

// Tag describes a structural label that can span several mass owners.
type Tag struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Family string `json:"family"` // arms, regions, satellites, tidal, nucleus...
	Note   string `json:"note"`
}
