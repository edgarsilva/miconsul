// Command featuremap renders docs/features.md, the verification plans and the
// Go tests into one static HTML page that shows what is covered and what is not.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	featuresFile   = "docs/features.md"
	plansDir       = ".agents/skills/verify-miconsul/plans"
	livesHeading   = "Where each feature lives"
	screensHeading = "Screens"
	flowsHeading   = "User flows"
	servicesDir    = "internal/services"
)

var (
	testName    = regexp.MustCompile("`(Test\\w+)`")
	testDecl    = regexp.MustCompile(`(?m)^func (Test\w+)\(`)
	planHeading = regexp.MustCompile(`(?m)^## \d+\.`)
	backticked  = regexp.MustCompile("`([^`]+)`")
)

type table struct {
	Heading string
	Rows    [][]string
}

type testRef struct {
	Name   string
	Exists bool
}

type feature struct {
	Name   string
	How    string
	Tests  []testRef
	OnFlow bool
}

type planRef struct {
	Name   string
	Steps  int
	Exists bool
}

type sourceRef struct {
	Path   string
	Exists bool
}

type area struct {
	Name     string
	Slug     string
	Features []feature
	Sources  []sourceRef
	Plans    []planRef
	Screens  int
	Entries  []string
	Flow     flowDiagram
}

type summary struct {
	Areas, WithPlan, WithoutPlan, Features, OffFlow, Screens, Tests, MissingTests, MissingPaths int
}

type report struct {
	Areas    []area
	Flow     flowDiagram
	Unmapped []string
	Summary  summary
}

func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "tmp/feature-map/index.html", "output file, relative to the root")
	flag.Parse()

	r, err := build(*root)
	if err != nil {
		log.Fatalf("build feature map: %v", err)
	}
	dest := filepath.Join(*root, *out)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		log.Fatalf("create output folder: %v", err)
	}
	f, err := os.Create(dest)
	if err != nil {
		log.Fatalf("create output file: %v", err)
	}
	defer f.Close()
	if err := page.Execute(f, r); err != nil {
		log.Fatalf("render feature map: %v", err)
	}
	s := r.Summary
	fmt.Printf("feature map: %s (%d areas, %d without a plan, %d missing tests, %d missing paths)\n",
		dest, s.Areas, s.WithoutPlan, s.MissingTests, s.MissingPaths)
}

func build(root string) (report, error) {
	md, err := os.ReadFile(filepath.Join(root, featuresFile))
	if err != nil {
		return report{}, fmt.Errorf("read features: %w", err)
	}
	tests, err := declaredTests(root)
	if err != nil {
		return report{}, err
	}
	steps, err := planSteps(root)
	if err != nil {
		return report{}, err
	}
	return assemble(root, parseTables(string(md)), tests, steps)
}

func assemble(root string, tables []table, tests map[string]bool, steps map[string]int) (report, error) {
	var features []struct {
		section string
		feature feature
	}
	var lives, screenRows, flowRows [][]string
	for _, t := range tables {
		switch t.Heading {
		case livesHeading:
			lives = t.Rows
			continue
		case screensHeading:
			screenRows = t.Rows
			continue
		case flowsHeading:
			flowRows = t.Rows
			continue
		}
		for _, row := range t.Rows {
			if len(row) < 3 {
				continue
			}
			f := feature{Name: row[0], How: strings.ReplaceAll(row[1], "`", "")}
			for _, m := range testName.FindAllStringSubmatch(row[2], -1) {
				f.Tests = append(f.Tests, testRef{Name: m[1], Exists: tests[m[1]]})
			}
			features = append(features, struct {
				section string
				feature feature
			}{t.Heading, f})
		}
	}

	var r report
	mapped := map[string]bool{}
	for _, row := range lives {
		if len(row) < 3 {
			continue
		}
		a := area{Name: row[0], Slug: slug(row[0])}
		for _, m := range backticked.FindAllStringSubmatch(row[1], -1) {
			p := m[1]
			a.Sources = append(a.Sources, sourceRef{Path: p, Exists: pathExists(root, p)})
			mapped[p] = true
		}
		for name := range strings.SplitSeq(row[2], ",") {
			name = strings.TrimSpace(name)
			if name == "" || name == "none yet" {
				continue
			}
			n, ok := steps[name]
			a.Plans = append(a.Plans, planRef{Name: name, Steps: n, Exists: ok})
		}
		for _, f := range features {
			if f.section == a.Name || f.feature.Name == a.Name {
				a.Features = append(a.Features, f.feature)
			}
		}
		r.Areas = append(r.Areas, a)
	}
	var err error
	if r.Flow, err = attachFlows(r.Areas, parseScreens(screenRows), parseScreenLinks(flowRows)); err != nil {
		return report{}, err
	}
	r.Unmapped = unmappedServices(root, mapped)
	r.Summary = summarize(r.Areas)
	return r, nil
}

func summarize(areas []area) summary {
	s := summary{Areas: len(areas)}
	for _, a := range areas {
		if len(a.Plans) > 0 {
			s.WithPlan++
		} else {
			s.WithoutPlan++
		}
		s.Screens += a.Screens
		for _, src := range a.Sources {
			if !src.Exists {
				s.MissingPaths++
			}
		}
		for _, f := range a.Features {
			s.Features++
			if !f.OnFlow {
				s.OffFlow++
			}
			for _, t := range f.Tests {
				s.Tests++
				if !t.Exists {
					s.MissingTests++
				}
			}
		}
	}
	return s
}

func parseTables(md string) []table {
	var tables []table
	var heading string
	for line := range strings.SplitSeq(md, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "## "):
			heading = strings.TrimPrefix(line, "## ")
			tables = append(tables, table{Heading: heading})
		case strings.HasPrefix(line, "|") && len(tables) > 0:
			cells := splitRow(line)
			if isHeaderOrRule(cells) {
				continue
			}
			t := &tables[len(tables)-1]
			t.Rows = append(t.Rows, cells)
		}
	}
	return tables
}

func splitRow(line string) []string {
	line = strings.Trim(line, "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func isHeaderOrRule(cells []string) bool {
	if len(cells) == 0 {
		return true
	}
	if strings.Trim(cells[0], "-: ") == "" {
		return true
	}
	return cells[0] == "Feature" || cells[0] == "Feature area" || cells[0] == "From" || cells[0] == "Screen"
}

func declaredTests(root string) (map[string]bool, error) {
	found := map[string]bool{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "tmp") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range testDecl.FindAllStringSubmatch(string(b), -1) {
			found[m[1]] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan tests: %w", err)
	}
	return found, nil
}

func planSteps(root string) (map[string]int, error) {
	files, err := filepath.Glob(filepath.Join(root, plansDir, "*.plan.md"))
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	steps := map[string]int{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read plan: %w", err)
		}
		name := strings.TrimSuffix(filepath.Base(f), ".plan.md")
		steps[name] = len(planHeading.FindAllString(string(b), -1))
	}
	return steps, nil
}

func pathExists(root, p string) bool {
	if strings.Contains(p, "*") {
		m, err := filepath.Glob(filepath.Join(root, p))
		return err == nil && len(m) > 0
	}
	_, err := os.Stat(filepath.Join(root, p))
	return err == nil
}

func unmappedServices(root string, mapped map[string]bool) []string {
	entries, err := os.ReadDir(filepath.Join(root, servicesDir))
	if err != nil {
		return nil
	}
	var missing []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		prefix := servicesDir + "/" + e.Name() + "/"
		if !mapped[prefix] {
			missing = append(missing, prefix)
		}
	}
	sort.Strings(missing)
	return missing
}

var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Miconsul feature map</title>
<style>
:root{--bg:#f7f7f5;--card:#fff;--ink:#1d1d1b;--mute:#6b6b66;--line:#e4e4df;--ok:#1f7a4d;--okbg:#e4f4ec;--warn:#9a5b00;--warnbg:#fdf0d8;--bad:#a52a2a;--badbg:#fbe4e4}
@media (prefers-color-scheme:dark){:root{--bg:#161614;--card:#1f1f1c;--ink:#ecece8;--mute:#9a9a93;--line:#33332f;--ok:#6fd1a0;--okbg:#17302a;--warn:#e6b25a;--warnbg:#33290f;--bad:#f09a9a;--badbg:#3a1c1c}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--ink);font:15px/1.5 system-ui,sans-serif}
main{max-width:1100px;margin:0 auto;padding:24px 16px 48px}
h1{margin:0 0 4px;font-size:1.6rem}
p.lead{margin:0 0 20px;color:var(--mute)}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:12px;margin-bottom:24px}
.stat{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:12px 14px}
.stat b{display:block;font-size:1.6rem}
.stat span{color:var(--mute);font-size:.85rem}
.stat.bad b{color:var(--bad)}.stat.warn b{color:var(--warn)}.stat.ok b{color:var(--ok)}
.area{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:16px;margin-bottom:14px}
.area header{display:flex;flex-wrap:wrap;gap:8px;align-items:center;margin-bottom:8px}
.area h2{margin:0;font-size:1.15rem;margin-right:auto}
.chip{display:inline-block;border-radius:999px;padding:1px 10px;font-size:.8rem;background:var(--okbg);color:var(--ok)}
.chip.warn{background:var(--warnbg);color:var(--warn)}.chip.bad{background:var(--badbg);color:var(--bad)}
ul{margin:6px 0;padding-left:18px}
li{margin:2px 0}
.how{color:var(--mute)}
code{font:.85rem ui-monospace,monospace}
.cols{display:grid;grid-template-columns:2fr 1fr;gap:16px}
@media (max-width:720px){.cols{grid-template-columns:1fr}}
h3{margin:8px 0 4px;font-size:.8rem;text-transform:uppercase;letter-spacing:.05em;color:var(--mute)}
.missing{color:var(--bad)}
.flow{overflow-x:auto}
.flow svg{width:100%;height:auto;min-width:760px}
.flow{margin:6px 0 10px}
.edge{fill:none;stroke:var(--mute);stroke-width:1.4;opacity:.7}
.arrow{fill:var(--mute)}
.edgelabel{font-size:11px;fill:var(--mute);paint-order:stroke;stroke:var(--card);stroke-width:4px}
.node rect{fill:var(--card);stroke-width:1.5}
.node.ok rect{stroke:var(--ok);fill:var(--okbg)}
.node.warn rect{stroke:var(--warn);fill:var(--warnbg)}
.node:hover rect{stroke-width:3}
.nname{font-size:14px;font-weight:600;fill:var(--ink)}
.ndetail{font-size:12px;fill:var(--mute)}
a{color:inherit}
</style>
</head>
<body>
<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs><marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" class="arrow"/></marker></defs></svg>
<main>
<h1>Miconsul feature map</h1>
<p class="lead">Built from docs/features.md, the verification plans and the Go tests. This is what each feature is proven by, not line coverage.</p>
<section class="stats">
<div class="stat"><b>{{.Summary.Areas}}</b><span>feature areas</span></div>
<div class="stat ok"><b>{{.Summary.WithPlan}}</b><span>with a verification plan</span></div>
<div class="stat {{if .Summary.WithoutPlan}}warn{{else}}ok{{end}}"><b>{{.Summary.WithoutPlan}}</b><span>without a plan</span></div>
<div class="stat"><b>{{.Summary.Screens}}</b><span>screens on user flows</span></div>
<div class="stat {{if .Summary.OffFlow}}warn{{else}}ok{{end}}"><b>{{.Summary.OffFlow}}</b><span>features not on a flow</span></div>
<div class="stat"><b>{{.Summary.Tests}}</b><span>tests named in the map</span></div>
<div class="stat {{if .Summary.MissingTests}}bad{{else}}ok{{end}}"><b>{{.Summary.MissingTests}}</b><span>named tests not found</span></div>
<div class="stat {{if .Summary.MissingPaths}}bad{{else}}ok{{end}}"><b>{{.Summary.MissingPaths}}</b><span>source paths not found</span></div>
</section>
<section class="area">
<h2>Overview: how areas connect</h2>
<p class="how">Boxes are feature areas, arrows are how someone gets from one to the next. Green has a verification plan, amber does not. Click a box to jump to its card. Each card below draws the screens inside that area.</p>
{{template "flowsvg" .Flow}}
{{if .Flow.Detached}}<p class="how">Not on the flow: {{range $i, $n := .Flow.Detached}}{{if $i}}, {{end}}{{$n}}{{end}}.</p>{{end}}
</section>
{{range .Areas}}
<section class="area" id="{{.Slug}}">
<header>
<h2>{{.Name}}</h2>
{{range .Plans}}<span class="chip {{if not .Exists}}bad{{end}}">plan: {{.Name}}{{if .Exists}} · {{.Steps}} steps{{else}} · file missing{{end}}</span>{{else}}<span class="chip warn">no plan yet</span>{{end}}
</header>
{{if .Entries}}<p class="how">Reached from {{range $i, $e := .Entries}}{{if $i}}; {{end}}{{$e}}{{end}}</p>{{end}}
{{if .Flow.Nodes}}<h3>User flow · green: has a test and a plan · amber: missing one</h3>
{{template "flowsvg" .Flow}}{{end}}
<div class="cols">
<div>
<h3>Features and their tests</h3>
{{if .Features}}<ul>{{range .Features}}
<li><b>{{.Name}}</b> <span class="how">— {{.How}}</span>
{{if .Tests}}<br>{{range .Tests}}<code {{if not .Exists}}class="missing" title="no such test"{{end}}>{{.Name}}</code> {{end}}{{else}}<br><span class="chip warn">no test named</span>{{end}}{{if not .OnFlow}} <span class="chip warn">not on a flow</span>{{end}}</li>
{{end}}</ul>{{else}}<p class="how">No feature rows; this area is shared infrastructure.</p>{{end}}
</div>
<div>
<h3>Source paths</h3>
<ul>{{range .Sources}}<li><code {{if not .Exists}}class="missing" title="path not found"{{end}}>{{.Path}}</code></li>{{end}}</ul>
</div>
</div>
</section>
{{end}}
<section class="area">
<h2>Services no area claims</h2>
{{if .Unmapped}}<ul>{{range .Unmapped}}<li><code>{{.}}</code></li>{{end}}</ul>{{else}}<p class="how">Every folder under internal/services belongs to an area.</p>{{end}}
</section>
</main>
</body>
</html>
`))

func init() {
	template.Must(page.Parse(`{{define "flowsvg"}}<div class="flow">
<svg viewBox="0 0 {{.Width}} {{.Height}}" role="img" aria-label="Flow">
{{range .Edges}}<path d="{{.Path}}" class="edge" marker-end="url(#arrow)"/><text x="{{.LabelX}}" y="{{.LabelY}}" class="edgelabel" text-anchor="{{if .Back}}middle{{else}}end{{end}}">{{.Label}}</text>
{{end}}
{{range .Nodes}}{{if .Href}}<a href="{{.Href}}">{{end}}<g class="node {{.Status}}">{{if .Tip}}<title>{{.Tip}}</title>{{end}}<rect x="{{.X}}" y="{{.Y}}" width="{{$.NodeWidth}}" height="{{$.NodeHeight}}" rx="8"/><text x="{{.X}}" dx="14" y="{{.Y}}" dy="22" class="nname">{{.Name}}</text><text x="{{.X}}" dx="14" y="{{.Y}}" dy="{{if gt $.NodeHeight 50}}44{{else}}39{{end}}" class="ndetail">{{.Detail}}</text></g>{{if .Href}}</a>{{end}}
{{end}}
</svg>
</div>{{end}}`))
}
