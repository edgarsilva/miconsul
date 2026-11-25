package main

import (
	"fmt"
	"strings"
)

type screen struct {
	Name, Area, Route, Feature string
}

type screenLink struct {
	From, Does, To string
}

func parseScreens(rows [][]string) []screen {
	out := make([]screen, 0, len(rows))
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		out = append(out, screen{Name: row[0], Area: row[1], Route: strings.ReplaceAll(row[2], "`", ""), Feature: row[3]})
	}
	return out
}

func parseScreenLinks(rows [][]string) []screenLink {
	out := make([]screenLink, 0, len(rows))
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		out = append(out, screenLink{From: row[0], Does: row[1], To: row[2]})
	}
	return out
}

func attachFlows(areas []area, screens []screen, links []screenLink) (flowDiagram, error) {
	areaIndex := map[string]int{}
	for i, a := range areas {
		areaIndex[a.Name] = i
	}

	screenIndex := map[string]int{}
	local := make([]int, len(screens))
	members := make([][]int, len(areas))
	for i, s := range screens {
		if _, dup := screenIndex[s.Name]; dup {
			return flowDiagram{}, fmt.Errorf("screen %q is listed twice", s.Name)
		}
		ai, ok := areaIndex[s.Area]
		if !ok {
			return flowDiagram{}, fmt.Errorf("screen %q names the area %q, which is not in %q", s.Name, s.Area, livesHeading)
		}
		screenIndex[s.Name] = i
		local[i] = len(members[ai])
		members[ai] = append(members[ai], i)
	}

	nodes := make([][]graphNode, len(areas))
	for ai, ids := range members {
		for _, si := range ids {
			s := screens[si]
			f := findFeature(areas, ai, s.Feature)
			if f == nil && s.Feature != "-" && s.Feature != "" {
				return flowDiagram{}, fmt.Errorf("screen %q names the feature %q, which is not in the feature tables", s.Name, s.Feature)
			}
			if f != nil {
				f.OnFlow = true
			}
			nodes[ai] = append(nodes[ai], screenNode(s, areas[ai], f))
		}
	}

	inArea := make([][]graphLink, len(areas))
	overview := map[[2]int]graphLink{}
	var overviewOrder [][2]int
	for _, l := range links {
		from, okFrom := screenIndex[l.From]
		to, okTo := screenIndex[l.To]
		if !okFrom || !okTo {
			return flowDiagram{}, fmt.Errorf("flow %q -> %q names a screen that is not in the screens table", l.From, l.To)
		}
		fa, ta := areaIndex[screens[from].Area], areaIndex[screens[to].Area]
		if fa == ta {
			inArea[fa] = append(inArea[fa], graphLink{From: local[from], To: local[to], Label: l.Does})
			continue
		}
		areas[ta].Entries = append(areas[ta].Entries, entryText(areas[fa].Name, l))
		key := [2]int{fa, ta}
		if _, seen := overview[key]; !seen {
			overview[key] = graphLink{From: fa, To: ta, Label: l.Does}
			overviewOrder = append(overviewOrder, key)
		}
	}

	for ai := range areas {
		d, err := layoutGraph(nodes[ai], inArea[ai], true, screenMetrics)
		if err != nil {
			return flowDiagram{}, fmt.Errorf("%s: %w", areas[ai].Name, err)
		}
		areas[ai].Flow = d
		areas[ai].Screens = len(nodes[ai])
	}

	areaNodes := make([]graphNode, len(areas))
	for i, a := range areas {
		areaNodes[i] = areaNode(a)
	}
	var overviewLinks []graphLink
	for _, k := range overviewOrder {
		overviewLinks = append(overviewLinks, overview[k])
	}
	d, err := layoutGraph(areaNodes, overviewLinks, false, overviewMetrics)
	if err != nil {
		return flowDiagram{}, fmt.Errorf("overview: %w", err)
	}
	return d, nil
}

func findFeature(areas []area, ai int, name string) *feature {
	if name == "" || name == "-" {
		return nil
	}
	for i := range areas[ai].Features {
		if areas[ai].Features[i].Name == name {
			return &areas[ai].Features[i]
		}
	}
	for a := range areas {
		for i := range areas[a].Features {
			if areas[a].Features[i].Name == name {
				return &areas[a].Features[i]
			}
		}
	}
	return nil
}

func entryText(areaName string, l screenLink) string {
	if areaName == l.From {
		return fmt.Sprintf("%s: %s", l.From, l.Does)
	}
	return fmt.Sprintf("%s › %s: %s", areaName, l.From, l.Does)
}
