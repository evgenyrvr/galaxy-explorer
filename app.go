package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/obsdata"
	"galaxy/internal/presets"
	"galaxy/internal/project"
)

// AppVersion is written into saved projects.
const AppVersion = "0.3.1"

const defaultSeed = 20261015

// App is the Wails-bound facade over the Go mass model. All mass logic lives
// in Go; the frontend only displays and requests changes.
type App struct {
	ctx      context.Context
	mu       sync.Mutex
	presets  map[string]galaxy.Preset
	model    *galaxy.Model
	datasets []obsdata.Dataset
}

// NewApp creates the application.
func NewApp() *App {
	a := &App{presets: map[string]galaxy.Preset{}}
	for _, p := range presets.All() {
		a.presets[p.Info().ID] = p
	}
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) current() (*galaxy.Model, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.model == nil {
		return nil, errors.New("no galaxy loaded")
	}
	return a.model, nil
}

// ListPresets returns the available galaxy templates. The first one is
// opened at start-up; GALAXY_PRESET=<id> selects a different one.
func (a *App) ListPresets() []galaxy.PresetInfo {
	var out []galaxy.PresetInfo
	first := os.Getenv("GALAXY_PRESET")
	for _, p := range presets.All() {
		info := a.presets[p.Info().ID].Info()
		if info.ID == first {
			out = append([]galaxy.PresetInfo{info}, out...)
		} else {
			out = append(out, info)
		}
	}
	return out
}

// LoadPreset opens a preset with its default budget.
func (a *App) LoadPreset(id string) (galaxy.State, error) {
	p, ok := a.presets[id]
	if !ok {
		return galaxy.State{}, fmt.Errorf("unknown preset %q", id)
	}
	m, err := galaxy.NewModel(p, defaultSeed)
	if err != nil {
		return galaxy.State{}, err
	}
	a.mu.Lock()
	a.model = m
	a.datasets = nil
	if dp, ok := p.(galaxy.DatasetProvider); ok {
		a.datasets = dp.Datasets() // bundled published tables
	}
	a.mu.Unlock()
	return m.Snapshot(), nil
}

// State returns the current snapshot.
func (a *App) State() (galaxy.State, error) {
	m, err := a.current()
	if err != nil {
		return galaxy.State{}, err
	}
	return m.Snapshot(), nil
}

func (a *App) edit(fn func(b *budget.Budget) error) (galaxy.State, error) {
	m, err := a.current()
	if err != nil {
		return galaxy.State{}, err
	}
	if err := m.Edit(fn); err != nil {
		return m.Snapshot(), err
	}
	return m.Snapshot(), nil
}

// SetTotalMass sets M200c (M☉) keeping all fractions.
func (a *App) SetTotalMass(mass float64) (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.SetTotalMass(mass) })
}

// SetFraction sets a category fraction (0..1) with a redistribution rule.
func (a *App) SetFraction(category string, fraction float64, rule string) (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.SetFraction(category, fraction, budget.Rule(rule)) })
}

// SetCategoryMass sets a category's absolute mass (M☉).
func (a *App) SetCategoryMass(category string, mass float64) (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.SetCategoryMass(category, mass) })
}

// SetLocked locks a category against redistribution.
func (a *App) SetLocked(category string, locked bool) (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.SetLocked(category, locked) })
}

// SetMode selects fix_total or fix_components.
func (a *App) SetMode(mode string) (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.SetMode(budget.Mode(mode)) })
}

// Normalize rescales unlocked categories to 100%.
func (a *App) Normalize() (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.Normalize() })
}

// AssignRemainder gives the unallocated fraction to one category.
func (a *App) AssignRemainder(category string) (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.AssignRemainder(category) })
}

// SetShare sets a component's share inside its category (0..1).
func (a *App) SetShare(category, component string, share float64) (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.SetShare(category, component, share) })
}

// SetShareLocked locks a component share.
func (a *App) SetShareLocked(category, component string, locked bool) (galaxy.State, error) {
	return a.edit(func(b *budget.Budget) error { return b.SetShareLocked(category, component, locked) })
}

// Undo reverts the last budget change.
func (a *App) Undo() (galaxy.State, error) {
	m, err := a.current()
	if err != nil {
		return galaxy.State{}, err
	}
	m.Undo()
	return m.Snapshot(), nil
}

// Redo re-applies an undone change.
func (a *App) Redo() (galaxy.State, error) {
	m, err := a.current()
	if err != nil {
		return galaxy.State{}, err
	}
	m.Redo()
	return m.Snapshot(), nil
}

// ResetBudget restores the preset defaults.
func (a *App) ResetBudget() (galaxy.State, error) {
	m, err := a.current()
	if err != nil {
		return galaxy.State{}, err
	}
	m.Reset()
	return m.Snapshot(), nil
}

// SelectionMass returns the single-counted mass of a selection.
func (a *App) SelectionMass(sel galaxy.Selection) (galaxy.SelectionReport, error) {
	m, err := a.current()
	if err != nil {
		return galaxy.SelectionReport{}, err
	}
	return m.SelectionMass(sel), nil
}

// Profiles returns radial profiles of the current model.
func (a *App) Profiles(req galaxy.ProfileRequest) (galaxy.Profiles, error) {
	m, err := a.current()
	if err != nil {
		return galaxy.Profiles{}, err
	}
	return m.Profiles(req), nil
}

// SaveProject asks for a path and writes the project; ui is the frontend's
// JSON document (camera, visibility, filters, selection).
func (a *App) SaveProject(ui string) (string, error) {
	m, err := a.current()
	if err != nil {
		return "", err
	}
	if ui != "" && !json.Valid([]byte(ui)) {
		return "", errors.New("UI state is not valid JSON")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Save project", DefaultFilename: "galaxy-project.galaxy.json",
		Filters: []runtime.FileFilter{{DisplayName: "Galaxy project (*.galaxy.json)", Pattern: "*.galaxy.json;*.json"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	a.mu.Lock()
	ds := append([]obsdata.Dataset(nil), a.datasets...)
	a.mu.Unlock()
	f := project.New(m, "Galaxy Explorer "+AppVersion, ds, json.RawMessage(ui))
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

// OpenResult is returned by OpenProject.
type OpenResult struct {
	Path  string       `json:"path"`
	State galaxy.State `json:"state"`
	UI    string       `json:"ui"`
}

// OpenProject asks for a project file and restores it.
func (a *App) OpenProject() (*OpenResult, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Open project",
		Filters: []runtime.FileFilter{{DisplayName: "Galaxy project (*.galaxy.json)", Pattern: "*.galaxy.json;*.json"}},
	})
	if err != nil || path == "" {
		return nil, err
	}
	return a.openPath(path)
}

// StartupProject opens the project given on the command line
// (galaxy.exe path	oile.galaxy.json), or returns nil.
func (a *App) StartupProject() (*OpenResult, error) {
	if len(os.Args) < 2 || strings.HasPrefix(os.Args[1], "-") {
		return nil, nil
	}
	return a.openPath(os.Args[1])
}

func (a *App) openPath(path string) (*OpenResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := project.Parse(data)
	if err != nil {
		return nil, err
	}
	p, ok := a.presets[f.PresetID]
	if !ok {
		return nil, fmt.Errorf("project uses unknown preset %q", f.PresetID)
	}
	m, err := galaxy.NewModel(p, f.Seed)
	if err != nil {
		return nil, err
	}
	if err := m.Replace(f.Budget, f.Seed); err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.model = m
	a.datasets = f.Datasets
	a.mu.Unlock()
	return &OpenResult{Path: path, State: m.Snapshot(), UI: string(f.UI)}, nil
}

// ExportMassCSV writes the mass table and the current selection.
func (a *App) ExportMassCSV(sel galaxy.Selection) (string, error) {
	m, err := a.current()
	if err != nil {
		return "", err
	}
	rep := m.SelectionMass(sel)
	var selp *galaxy.SelectionReport
	if len(sel.Components)+len(sel.Tags) > 0 || sel.Region.Active() {
		selp = &rep
	}
	data, err := project.MassCSV(m.Snapshot(), selp, sel.Region)
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Export mass table", DefaultFilename: "galaxy-masses.csv",
		Filters: []runtime.FileFilter{{DisplayName: "CSV (*.csv)", Pattern: "*.csv"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

// SaveScreenshot writes a PNG given as a data URL.
func (a *App) SaveScreenshot(dataURL string) (string, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		return "", errors.New("screenshot is not a PNG data URL")
	}
	png, err := base64.StdEncoding.DecodeString(dataURL[len(prefix):])
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Save screenshot", DefaultFilename: "galaxy.png",
		Filters: []runtime.FileFilter{{DisplayName: "PNG image (*.png)", Pattern: "*.png"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, png, 0o644)
}

// particleMiddleware serves binary marker buffers at /api/particles?id=&n=,
// avoiding JSON for hundreds of thousands of floats.
func (a *App) particleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/particles" {
			next.ServeHTTP(w, r)
			return
		}
		m, err := a.current()
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		buf, err := m.Particles(r.URL.Query().Get("id"), n)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(buf)
	})
}

// ImportProfile asks for an observational profile CSV and adds it.
func (a *App) ImportProfile() ([]obsdata.Dataset, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Import radial profile",
		Filters: []runtime.FileFilter{{DisplayName: "Profile CSV (*.csv)", Pattern: "*.csv;*.txt"}},
	})
	if err != nil || path == "" {
		return a.Datasets(), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return a.Datasets(), err
	}
	d, err := obsdata.Parse(filepath.Base(path), data)
	if err != nil {
		return a.Datasets(), fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	d.ID = fmt.Sprintf("ds%d", len(a.datasets)+1)
	for _, x := range a.datasets {
		if x.ID == d.ID {
			d.ID += "b"
		}
	}
	a.datasets = append(a.datasets, d)
	return append([]obsdata.Dataset(nil), a.datasets...), nil
}

// Datasets returns the imported profiles.
func (a *App) Datasets() []obsdata.Dataset {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]obsdata.Dataset(nil), a.datasets...)
}

// RemoveDataset deletes an imported profile.
func (a *App) RemoveDataset(id string) []obsdata.Dataset {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.datasets[:0]
	for _, d := range a.datasets {
		if d.ID != id {
			out = append(out, d)
		}
	}
	a.datasets = out
	return append([]obsdata.Dataset(nil), a.datasets...)
}

// CompareProfiles returns the profiles of another preset with its default
// fractions at the current M_total, for overlays on the same axes.
func (a *App) CompareProfiles(presetID string) (galaxy.Profiles, error) {
	cur, err := a.current()
	if err != nil {
		return galaxy.Profiles{}, err
	}
	p, ok := a.presets[presetID]
	if !ok {
		return galaxy.Profiles{}, fmt.Errorf("unknown preset %q", presetID)
	}
	m, err := galaxy.NewModel(p, defaultSeed)
	if err != nil {
		return galaxy.Profiles{}, err
	}
	total := cur.Budget().TotalMass
	if err := m.Edit(func(b *budget.Budget) error { return b.SetTotalMass(total) }); err != nil {
		return galaxy.Profiles{}, err
	}
	return m.Profiles(galaxy.ProfileRequest{RMax: cur.Snapshot().R200c}), nil
}
