package ui

import "testing"

// TestQuestDataRem: lines and nodes arrive in rem (place * 7.2), vias are not
// nodes, the focus is in rem and the open chapter has no id.
func TestQuestDataRem(t *testing.T) {
	var root any
	oreShown = func(_ *oreLayout, r any) { root = r }
	defer func() { oreShown = nil }()

	QuestBookMenu(nil, "q", QuestBook{
		Chapters: []QuestChapter{{Name: "a", On: true, Run: nopRun}, {Name: "b", Run: nopRun}},
		Nodes: []QuestNode{
			{ID: "a", X: 0, Y: 1, Run: nopRun},
			{ID: "b", X: 1, Y: 1, Deps: []string{"a"}, Run: nopRun},
			{ID: "v", X: 2, Y: 1, Shape: ShapeVia},
		},
		Focus: "b",
	})
	name, data := screenRoot(root)
	if name != "quest" || data["heading"] != "q" || data["focusX"] != 7.2 || data["focusY"] != 7.2 {
		t.Fatalf("%s %v", name, data)
	}
	nodes := data["nodes"].([]any)
	if len(nodes) != 2 || nodes[1].(map[string]any)["x"] != 7.2 || nodes[0].(map[string]any)["shape"] != "square" {
		t.Errorf("nodes %v", nodes)
	}
	line := data["lines"].([]any)[0].(map[string]any)
	if line["x1"] != 0.0 || line["x2"] != 7.2 || line["y1"] != 7.2 {
		t.Errorf("line %v", line)
	}
	ch := data["chapters"].([]any)
	if ch[0].(map[string]any)["id"] != "" || ch[1].(map[string]any)["id"] == "" || data["panel"] != nil {
		t.Errorf("chapters %v panel %v", ch, data["panel"])
	}
}

// TestQuestNew: a new quest carries its NEW tag and the id its look sends, an
// old one neither; Recenter and a chapter's New go through.
func TestQuestNew(t *testing.T) {
	var root any
	oreShown = func(_ *oreLayout, r any) { root = r }
	defer func() { oreShown = nil }()

	QuestBookMenu(nil, "q", QuestBook{
		Chapters: []QuestChapter{{Name: "a", On: true, New: true}},
		Nodes: []QuestNode{
			{ID: "a", Run: nopRun, Seen: nopRun},
			{ID: "b", Run: nopRun, New: true, Seen: nopRun},
		},
		Focus: "b", Recenter: true,
	})
	_, data := screenRoot(root)
	nodes := data["nodes"].([]any)
	a, b := nodes[0].(map[string]any), nodes[1].(map[string]any)
	if a["new"] != false || a["seen"] != "" || b["new"] != true || b["seen"] == "" || b["seen"] == b["id"] {
		t.Errorf("nodes %v", nodes)
	}
	if data["recenter"] != true || data["chapters"].([]any)[0].(map[string]any)["new"] != true {
		t.Errorf("recenter %v chapters %v", data["recenter"], data["chapters"])
	}
}
