package main

import (
	"fmt"
	"sort"
	"strings"
)

const (
	marginX = 16
	marginY = 16
)

type metrics struct {
	nodeWidth, nodeHeight, gapY, gapX int
}

var (
	overviewMetrics = metrics{nodeWidth: 250, nodeHeight: 58, gapY: 16, gapX: 110}
	screenMetrics   = metrics{nodeWidth: 210, nodeHeight: 48, gapY: 14, gapX: 130}
)

type graphNode struct {
	Name, Detail, Tip, Status, Href string
}

type graphLink struct {
	From, To int
	Label    string
}

type flowNode struct {
	graphNode
	X, Y int
}

type flowEdge struct {
	Path, Label    string
	LabelX, LabelY int
}

type flowDiagram struct {
	Width, Height, NodeWidth, NodeHeight int
	Nodes                                []flowNode
	Edges                                []flowEdge
	Detached                             []string
}

func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func layoutGraph(nodes []graphNode, links []graphLink, keepLone bool, m metrics) (flowDiagram, error) {
	connected := map[int]bool{}
	for _, l := range links {
		connected[l.From], connected[l.To] = true, true
	}

	layer := make([]int, len(nodes))
	for pass := 0; ; pass++ {
		if pass > len(nodes) {
			return flowDiagram{}, fmt.Errorf("flow has a cycle")
		}
		changed := false
		for _, l := range links {
			if layer[l.To] < layer[l.From]+1 {
				layer[l.To] = layer[l.From] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	var d flowDiagram
	var columns [][]int
	for i, n := range nodes {
		if !connected[i] && !keepLone {
			d.Detached = append(d.Detached, n.Name)
			continue
		}
		for len(columns) <= layer[i] {
			columns = append(columns, nil)
		}
		columns[layer[i]] = append(columns[layer[i]], i)
	}
	if len(columns) == 0 {
		return d, nil
	}

	rank := map[int]int{}
	for col, members := range columns {
		if col > 0 {
			sortByParents(members, links, rank)
		}
		for row, i := range members {
			rank[i] = row
		}
	}

	tallest := 0
	for _, c := range columns {
		tallest = max(tallest, len(c))
	}
	d.NodeWidth, d.NodeHeight = m.nodeWidth, m.nodeHeight
	d.Width = 2*marginX + len(columns)*m.nodeWidth + (len(columns)-1)*m.gapX
	d.Height = 2*marginY + tallest*(m.nodeHeight+m.gapY) - m.gapY

	position := map[int][2]int{}
	for col, members := range columns {
		x := marginX + col*(m.nodeWidth+m.gapX)
		top := (d.Height - (len(members)*(m.nodeHeight+m.gapY) - m.gapY)) / 2
		for row, i := range members {
			y := top + row*(m.nodeHeight+m.gapY)
			position[i] = [2]int{x, y}
			d.Nodes = append(d.Nodes, flowNode{graphNode: nodes[i], X: x, Y: y})
		}
	}

	bend := m.gapX / 2
	for _, l := range links {
		from, to := position[l.From], position[l.To]
		x1, y1 := from[0]+m.nodeWidth, from[1]+m.nodeHeight/2
		x2, y2 := to[0], to[1]+m.nodeHeight/2
		d.Edges = append(d.Edges, flowEdge{
			Path:   fmt.Sprintf("M %d %d C %d %d, %d %d, %d %d", x1, y1, x1+bend, y1, x2-bend, y2, x2, y2),
			Label:  l.Label,
			LabelX: x2 - 8,
			LabelY: y2 - 6,
		})
	}
	return d, nil
}

func areaNode(a area) graphNode {
	tests := 0
	for _, f := range a.Features {
		tests += len(f.Tests)
	}
	detail := fmt.Sprintf("%d tests · no plan yet", tests)
	status := "warn"
	switch len(a.Plans) {
	case 0:
	case 1:
		detail, status = fmt.Sprintf("%d tests · 1 plan", tests), "ok"
	default:
		detail, status = fmt.Sprintf("%d tests · %d plans", tests, len(a.Plans)), "ok"
	}
	return graphNode{Name: a.Name, Detail: detail, Status: status, Href: "#" + a.Slug}
}

func screenNode(s screen, owner area, f *feature) graphNode {
	n := graphNode{Name: s.Name, Detail: s.Route, Status: "warn", Tip: "no feature row names this screen"}
	if f == nil {
		return n
	}
	n.Tip = fmt.Sprintf("%s · %d tests", f.Name, len(f.Tests))
	if len(f.Tests) > 0 && len(owner.Plans) > 0 {
		n.Status = "ok"
	}
	return n
}

func sortByParents(members []int, links []graphLink, rank map[int]int) {
	key := map[int]float64{}
	for _, i := range members {
		sum, n := 0, 0
		for _, l := range links {
			if l.To == i {
				sum += rank[l.From]
				n++
			}
		}
		if n > 0 {
			key[i] = float64(sum) / float64(n)
		}
	}
	sort.SliceStable(members, func(a, b int) bool { return key[members[a]] < key[members[b]] })
}
