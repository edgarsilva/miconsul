package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleFeatures = `# Features

## Accounts

| Feature | How the user reaches it | Proof |
|---|---|---|
| Sign in | the form | Lands on the dashboard. Tests: ` + "`TestKnown`, `TestGone`" + ` |

## Revenue and profile

| Feature | How the user reaches it | Proof |
|---|---|---|
| Revenue | the navbar | Totals. Tests: ` + "`TestKnown`" + ` |

## Where each feature lives

| Feature area | Source paths | Plan |
|---|---|---|
| Accounts | ` + "`internal/services/auth/`, `nowhere/`" + ` | sign-in |
| Revenue | ` + "`internal/services/revenue/`" + ` | none yet |

## Screens

| Screen | Area | Route | Feature |
|---|---|---|---|
| Sign in | Accounts | /signin | Sign in |
| Totals | Revenue | /revenue | Revenue |

## User flows

| From | Does | To |
|---|---|---|
| Sign in | signs in | Totals |
`

func TestAssembleMapsFeaturesPlansAndGaps(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"internal/services/auth", "internal/services/revenue", "internal/services/orphan"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	tests := map[string]bool{"TestKnown": true}
	steps := map[string]int{"sign-in": 4}

	r, err := assemble(root, parseTables(sampleFeatures), tests, steps)
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Areas) != 2 {
		t.Fatalf("areas = %d, want 2", len(r.Areas))
	}
	accounts, revenue := r.Areas[0], r.Areas[1]
	if len(accounts.Features) != 1 || len(accounts.Features[0].Tests) != 2 {
		t.Fatalf("accounts features = %+v", accounts.Features)
	}
	if accounts.Features[0].Tests[1].Exists {
		t.Fatal("TestGone should be reported missing")
	}
	if len(accounts.Plans) != 1 || accounts.Plans[0].Steps != 4 {
		t.Fatalf("accounts plans = %+v", accounts.Plans)
	}
	if len(revenue.Features) != 1 || revenue.Features[0].Name != "Revenue" {
		t.Fatalf("revenue should pick up its row from a differently named section: %+v", revenue.Features)
	}
	if len(revenue.Plans) != 0 {
		t.Fatalf("revenue plans = %+v, want none", revenue.Plans)
	}
	want := summary{Areas: 2, WithPlan: 1, WithoutPlan: 1, Features: 2, Screens: 2, Tests: 3, MissingTests: 1, MissingPaths: 1}
	if r.Summary != want {
		t.Fatalf("summary = %+v, want %+v", r.Summary, want)
	}
	if len(r.Unmapped) != 1 || r.Unmapped[0] != "internal/services/orphan/" {
		t.Fatalf("unmapped = %v", r.Unmapped)
	}
}

func TestBuildAgainstTheRepository(t *testing.T) {
	r, err := build("../..")
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary.MissingTests != 0 {
		t.Fatalf("docs/features.md names %d tests that do not exist", r.Summary.MissingTests)
	}
	if r.Summary.MissingPaths != 0 {
		t.Fatalf("docs/features.md maps %d source paths that do not exist", r.Summary.MissingPaths)
	}
	if len(r.Unmapped) != 0 {
		t.Fatalf("services without an area: %s", strings.Join(r.Unmapped, ", "))
	}
}

func TestLayoutGraphPlacesTargetsInLaterColumns(t *testing.T) {
	nodes := []graphNode{{Name: "A", Status: "warn"}, {Name: "B", Status: "ok"}, {Name: "C"}, {Name: "Alone"}}
	links := []graphLink{{From: 0, To: 1, Label: "x"}, {From: 1, To: 2, Label: "y"}, {From: 0, To: 2, Label: "z"}}

	d, err := layoutGraph(nodes, links, false, overviewMetrics)
	if err != nil {
		t.Fatal(err)
	}
	x := map[string]int{}
	for _, n := range d.Nodes {
		x[n.Name] = n.X
	}
	if !(x["A"] < x["B"] && x["B"] < x["C"]) {
		t.Fatalf("columns out of order: %v", x)
	}
	if len(d.Edges) != 3 {
		t.Fatalf("edges = %d, want 3", len(d.Edges))
	}
	if len(d.Detached) != 1 || d.Detached[0] != "Alone" {
		t.Fatalf("detached = %v", d.Detached)
	}

	kept, err := layoutGraph(nodes, links, true, screenMetrics)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept.Nodes) != 4 || len(kept.Detached) != 0 {
		t.Fatalf("lone nodes should stay: nodes=%d detached=%v", len(kept.Nodes), kept.Detached)
	}
}

func TestLayoutGraphRejectsCycles(t *testing.T) {
	nodes := []graphNode{{Name: "A"}, {Name: "B"}}
	if _, err := layoutGraph(nodes, []graphLink{{From: 0, To: 1}, {From: 1, To: 0}}, false, overviewMetrics); err == nil {
		t.Fatal("cycle should fail")
	}
}

func TestAttachFlowsRoutesScreensIntoAreasAndOverview(t *testing.T) {
	areas := []area{
		{Name: "Accounts", Plans: []planRef{{Name: "sign-in"}}, Features: []feature{{Name: "Sign in", Tests: []testRef{{Name: "T", Exists: true}}}}},
		{Name: "Dashboard", Features: []feature{{Name: "Dashboard"}}},
	}
	screens := []screen{
		{Name: "Sign in", Area: "Accounts", Route: "/signin", Feature: "Sign in"},
		{Name: "Dashboard", Area: "Dashboard", Route: "/dashboard", Feature: "Dashboard"},
		{Name: "Filter", Area: "Dashboard", Route: "/dashboard/x", Feature: "-"},
	}
	links := []screenLink{{"Sign in", "signs in", "Dashboard"}, {"Dashboard", "tabs", "Filter"}}

	overview, err := attachFlows(areas, screens, links)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Nodes) != 2 || len(overview.Edges) != 1 {
		t.Fatalf("overview nodes=%d edges=%d", len(overview.Nodes), len(overview.Edges))
	}
	if len(areas[1].Entries) != 1 || !strings.Contains(areas[1].Entries[0], "Sign in") {
		t.Fatalf("dashboard entries = %v", areas[1].Entries)
	}
	if len(areas[1].Flow.Nodes) != 2 || len(areas[1].Flow.Edges) != 1 {
		t.Fatalf("dashboard flow = %+v", areas[1].Flow)
	}
	if !areas[0].Features[0].OnFlow || areas[0].Flow.Nodes[0].Status != "ok" {
		t.Fatalf("accounts flow = %+v", areas[0].Flow.Nodes)
	}
	if areas[1].Flow.Nodes[0].Status != "warn" {
		t.Fatalf("a feature with no test and no plan should be amber: %+v", areas[1].Flow.Nodes[0])
	}
}

func TestAttachFlowsRejectsUnknownReferences(t *testing.T) {
	areas := []area{{Name: "A", Features: []feature{{Name: "F"}}}}
	cases := map[string]struct {
		screens []screen
		links   []screenLink
	}{
		"unknown area":    {[]screen{{Name: "S", Area: "Nope", Feature: "-"}}, nil},
		"unknown feature": {[]screen{{Name: "S", Area: "A", Feature: "Nope"}}, nil},
		"duplicate":       {[]screen{{Name: "S", Area: "A", Feature: "-"}, {Name: "S", Area: "A", Feature: "-"}}, nil},
		"unknown screen":  {[]screen{{Name: "S", Area: "A", Feature: "-"}}, []screenLink{{"S", "x", "Ghost"}}},
	}
	for name, c := range cases {
		if _, err := attachFlows(areas, c.screens, c.links); err == nil {
			t.Errorf("%s should fail", name)
		}
	}
}

func TestEntryTextDropsAnAreaThatSharesTheScreenName(t *testing.T) {
	l := screenLink{"Dashboard", "navbar", "Clinics list"}
	if got := entryText("Dashboard", l); got != "Dashboard: navbar" {
		t.Fatalf("same name: %q", got)
	}
	if got := entryText("Home", l); got != "Home › Dashboard: navbar" {
		t.Fatalf("different name: %q", got)
	}
}
