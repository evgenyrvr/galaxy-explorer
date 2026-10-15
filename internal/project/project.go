// Package project defines the on-disk project format (.galaxy.json) and the
// mass CSV export.
package project

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/units"
)

// Schema identifies the file format.
const (
	Schema  = "galaxy-explorer/project"
	Version = 1
)

// File is a saved project. Physics (budget, seed, cosmology, preset version)
// is owned by Go; UI holds camera, visibility, filters and selection as an
// opaque document owned by the frontend.
type File struct {
	Schema         string          `json:"schema"`
	Version        int             `json:"version"`
	App            string          `json:"app"`
	SavedAt        string          `json:"savedAt"`
	PresetID       string          `json:"presetId"`
	PresetVersion  string          `json:"presetVersion"`
	DataMode       string          `json:"dataMode"`
	HaloDefinition string          `json:"haloDefinition"`
	Cosmology      units.Cosmology `json:"cosmology"`
	Seed           int64           `json:"seed"`
	Budget         budget.Budget   `json:"budget"`
	Sources        []string        `json:"sources"`
	UI             json.RawMessage `json:"ui,omitempty"`
}

// New builds a project file from a model.
func New(m *galaxy.Model, app string, ui json.RawMessage) File {
	s := m.Snapshot()
	var src []string
	for _, x := range s.Sources {
		src = append(src, x.ID)
	}
	return File{
		Schema: Schema, Version: Version, App: app, SavedAt: time.Now().UTC().Format(time.RFC3339),
		PresetID: s.Preset.ID, PresetVersion: s.Preset.Version, DataMode: s.Preset.Kind,
		HaloDefinition: "M200c", Cosmology: s.Cosmology, Seed: m.Seed(), Budget: m.Budget(), Sources: src, UI: ui,
	}
}

// Parse decodes and checks a project file.
func Parse(data []byte) (File, error) {
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return f, fmt.Errorf("not a valid project file: %w", err)
	}
	if f.Schema != Schema {
		return f, fmt.Errorf("unexpected schema %q", f.Schema)
	}
	if f.Version > Version {
		return f, fmt.Errorf("project version %d is newer than this program supports (%d)", f.Version, Version)
	}
	if f.HaloDefinition != "" && f.HaloDefinition != "M200c" {
		return f, fmt.Errorf("halo definition %q is not supported yet", f.HaloDefinition)
	}
	return f, nil
}

// MassCSV renders the component masses and an optional selection summary.
func MassCSV(s galaxy.State, sel *galaxy.SelectionReport) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	g := func(x float64) string { return strconv.FormatFloat(x, 'g', 10, 64) }
	_ = w.Write([]string{"# Galaxy Explorer mass table", s.Preset.Name, "mode=" + s.Preset.Kind, "halo=M200c", "cosmology=" + s.Cosmology.Name})
	_ = w.Write([]string{"# M_total_msun", g(s.TotalMass), "R200c_kpc", g(s.R200c), "unallocated_fraction", g(s.Unallocated)})
	_ = w.Write([]string{"component_id", "name", "matter", "structure", "evidence", "mass_msun", "fraction_of_total", "share_of_category", "inside_r200c_fraction", "sources"})
	for _, c := range s.Components {
		src := ""
		for i, x := range c.Sources {
			if i > 0 {
				src += ";"
			}
			src += x
		}
		_ = w.Write([]string{c.ID, c.Name, c.Matter, c.Structure, c.Evidence, g(c.Mass), g(c.FractionOfTotal), g(c.Share), g(c.InsideR200c), src})
	}
	if sel != nil {
		_ = w.Write(nil)
		_ = w.Write([]string{"# selection", "mass_msun", g(sel.Mass), "fraction_of_total", g(sel.FractionOfTotal), "exact", strconv.FormatBool(sel.Exact)})
		_ = w.Write([]string{"selection_component", "matter", "fraction_of_component", "mass_msun"})
		for _, p := range sel.Parts {
			_ = w.Write([]string{p.Component, p.Matter, g(p.Fraction), g(p.Mass)})
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
