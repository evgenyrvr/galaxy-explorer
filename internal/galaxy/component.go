// Package galaxy assembles presets, the mass budget and spatial
// distributions into a queryable galaxy model.
package galaxy

import (
	"galaxy/internal/budget"
	"galaxy/internal/obsdata"
	"galaxy/internal/ontology"
	"galaxy/internal/physics"
	"galaxy/internal/units"
)

// Visual is scenographic styling; it never affects physics.
type Visual struct {
	Style   string  `json:"style"` // stars | gas | dust | dm | icon | jet | clusters
	Color   string  `json:"color"`
	Size    float64 `json:"size"`    // sprite size in kpc (world units)
	Opacity float64 `json:"opacity"` // 0..1
	Markers int     `json:"markers"` // default number of visual markers
}

// Component is one mass owner: a (matter category, structure) pair with its
// own spatial distribution.
type Component struct {
	ID        string
	Name      string
	Matter    ontology.MatterType
	Structure string   // structural region, e.g. thin_disk
	Tags      []string // whole-owner structural tags, e.g. satellites
	Evidence  ontology.Evidence
	Sources   []string
	Notes     string
	Dist      physics.Distribution
	Visual    Visual
	// GeomKey changes whenever the geometry changes; the renderer refetches markers.
	GeomKey string
}

// Env is the physical context a preset is built in.
type Env struct {
	TotalMass float64 // M200c, M☉
	R200c     float64 // kpc
	Cosmology units.Cosmology
	Seed      int64
	// EnclosedMass of the preset's default mass model, for orbit-based
	// structures (streams, shells). Nil during the first build pass.
	DefaultEnclosed physics.EnclosedMass
}

// Preset is a galaxy template.
type Preset interface {
	Info() PresetInfo
	// DefaultBudget returns the literature-motivated default budget.
	DefaultBudget() budget.Budget
	// Build constructs the components for an environment. Geometry is fixed
	// ("geometry fixed" mode) except for quantities tied to the halo
	// boundary, which follow R200c.
	Build(env Env) []*Component
	// Tags describes the structural tags used by the preset.
	Tags() []ontology.Tag
}

// DatasetProvider is implemented by presets that bundle published tables.
type DatasetProvider interface {
	Datasets() []obsdata.Dataset
}

// PresetInfo is shown in the preset picker.
type PresetInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Morphology  string `json:"morphology"` // spiral | elliptical
	Kind        string `json:"kind"`       // normalized | observational
	Version     string `json:"version"`
	Description string `json:"description"`
	Disclaimer  string `json:"disclaimer"`
}
