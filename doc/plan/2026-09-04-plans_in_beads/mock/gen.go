//go:build ignore

// Command gen builds the fixtures for the "Issues" tab presentation mock in
// web/mock/plans_in_beads.
//
// Run it from the module root:
//
//	go run doc/plan/2026-09-04-plans_in_beads/mock/gen.go
//
// It reads the checked-in beads export beside this file (issues-export.jsonl,
// a copy of `bd export --all` taken 2026-09-04), synthesizes the plan issue
// this plan directory would become once D1's convention is applied (idea in
// description, plan in design, success criteria in acceptance, status in
// notes, decisions as comments, implementation steps as child tasks), adds
// the dependency edges the graph view draws (D14, D15), renders every
// markdown field through the previewer's own renderer
// (crabswarm/preview/render) and writes:
//
//	web/mock/plans_in_beads/api/fixtures.json                the mock's data
//	web/mock/plans_in_beads/public/assets/css/alert.css      served at the
//	web/mock/plans_in_beads/public/assets/css/chroma-*.css   paths signals/ui.ts links
//
// fixtures.json is shaped like the messages of PLAN.md's proto sketch
// (ngicks.crabswarm.issues.v1: Source, IssueSummary, RenderedField,
// IssueComment, IssueDependency, Issue, IssueEdge) in protobuf-JSON spelling: camelCase
// field names, ISSUE_STATUS_* enum names, RFC 3339 timestamps. Deviations from
// that sketch are listed in web/mock/plans_in_beads/MOCK_LIMITS.md.
//
// Nothing here talks to bd or to a daemon; the mock is a static fixture.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/ngicks/crabswarm/crabswarm/preview/render"
	"github.com/ngicks/crabswarm/crabswarm/preview/render/alert"
)

// planID is the ID the dogfood plan issue (PLAN.md step 10) would get.
const planID = "crabswarm-plan1"

// ideaGate is D7's metadata value: the date the idea gate was confirmed.
const ideaGate = "2026-09-04"

// agentsPlanID is the plan issue invented for the second source, so that
// source has an epic lane on the board too.
const agentsPlanID = "agents-package-p1"

// discoveredFrom are two real backlog issues presented as born from this plan
// (D1 / UC4). Both are referenced by the plan text itself.
var discoveredFrom = []string{"crabswarm-d2j", "crabswarm-aki"}

// --- fixture shapes (protobuf-JSON spelling of PLAN.md's issues/v1) ---------

type fixtures struct {
	Sources []source `json:"sources"`
	Issues  []issue  `json:"issues"`
	// Edges is what ListDependencies would return per source, flattened
	// across sources the way Issues is.
	Edges []edge `json:"edges"`
}

// edge mirrors issues.v1.IssueEdge, plus sourceId. From is the dependent,
// the child or the discoverer; To is the blocker, the parent or the origin.
type edge struct {
	SourceID string `json:"sourceId"`
	FromID   string `json:"fromId"`
	ToID     string `json:"toId"`
	Type     string `json:"type"`
}

// source mirrors issues.v1.Source.
type source struct {
	ID        string `json:"id"`
	Prefix    string `json:"prefix"`
	BeadsPath string `json:"beadsPath"`
	Dir       string `json:"dir"`
}

// heading mirrors preview.v1.Heading (RenderedField.toc).
type heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	ID    string `json:"id"`
}

// renderedField mirrors issues.v1.RenderedField.
type renderedField struct {
	HTML string    `json:"html"`
	TOC  []heading `json:"toc"`
}

// summary mirrors issues.v1.IssueSummary.
type summary struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	IssueType    string   `json:"issueType"`
	Status       string   `json:"status"`
	Priority     int      `json:"priority"`
	Labels       []string `json:"labels"`
	ParentID     string   `json:"parentId"`
	CommentCount int      `json:"commentCount"`
	ChildCount   int      `json:"childCount"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
	// ChildClosedCount feeds the epic progress affordance (D14).
	ChildClosedCount int `json:"childClosedCount"`
	// MetadataJSON on the summary lets the list and board show chips.
	MetadataJSON string `json:"metadataJson"`
}

// comment mirrors issues.v1.IssueComment.
type comment struct {
	ID        string        `json:"id"`
	Author    string        `json:"author"`
	Text      renderedField `json:"text"`
	CreatedAt string        `json:"createdAt"`
}

// dependency mirrors issues.v1.IssueDependency. Outgoing is true when this
// issue is the edge's From: it depends on, is a child of, or was discovered
// from the other issue.
type dependency struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Outgoing bool   `json:"outgoing"`
}

// issue mirrors issues.v1.Issue, plus sourceId: the fixture is one flat list
// across both sources, where the RPC would carry the source in the request.
type issue struct {
	SourceID           string        `json:"sourceId"`
	Summary            summary       `json:"summary"`
	Description        renderedField `json:"description"`
	Design             renderedField `json:"design"`
	AcceptanceCriteria renderedField `json:"acceptanceCriteria"`
	Notes              renderedField `json:"notes"`
	MetadataJSON       string        `json:"metadataJson"`
	// CloseReason is rendered here; the proto sketch has it as a plain string.
	CloseReason renderedField `json:"closeReason"`
	Comments    []comment     `json:"comments"`
	Children    []summary     `json:"children"`
	Depends     []dependency  `json:"dependencies"`
}

// --- bd export shapes -------------------------------------------------------

type exportRecord struct {
	ID                 string          `json:"id"`
	Title              string          `json:"title"`
	Description        string          `json:"description"`
	Design             string          `json:"design"`
	AcceptanceCriteria string          `json:"acceptance_criteria"`
	Notes              string          `json:"notes"`
	CloseReason        string          `json:"close_reason"`
	Status             string          `json:"status"`
	Priority           int             `json:"priority"`
	IssueType          string          `json:"issue_type"`
	Labels             []string        `json:"labels"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
	Comments           []exportComment `json:"comments"`
	CommentCount       int             `json:"comment_count"`
}

type exportComment struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

func main() {
	log.SetFlags(0)

	dirs := locate()
	renderer := render.New(render.Options{})
	r := &fieldRenderer{renderer: renderer}

	records, err := readExport(dirs.mock)
	if err != nil {
		log.Fatalf("read export: %v", err)
	}

	crabswarm := source{
		Prefix:    "crabswarm",
		BeadsPath: "/home/watage/gitrepo/github.com/ngicks/crabswarm/.beads",
		Dir:       dirs.module,
	}
	crabswarm.ID = sourceID(crabswarm.BeadsPath)
	agents := source{
		Prefix:    "agents-package",
		BeadsPath: "/home/watage/gitrepo/github.com/ngicks/agents-package/.beads",
		Dir:       "/home/watage/gitrepo/github.com/ngicks/agents-package",
	}
	agents.ID = sourceID(agents.BeadsPath)

	backlog := backlogIssues(r, crabswarm.ID, records)
	plan, steps, err := planIssues(r, crabswarm.ID, dirs.plan)
	if err != nil {
		log.Fatalf("build plan issue: %v", err)
	}

	issues := make([]issue, 0, len(backlog)+len(steps)+4)
	issues = append(issues, plan)
	issues = append(issues, steps...)
	issues = append(issues, backlog...)
	issues = append(issues, agentsIssues(r, agents.ID)...)

	edges := make([]edge, 0, 32)
	edges = append(edges, planEdges(crabswarm.ID, len(steps))...)
	edges = append(edges, backlogEdges(crabswarm.ID)...)
	edges = append(edges, agentsEdges(agents.ID)...)
	if err := link(issues, edges); err != nil {
		log.Fatalf("link issues: %v", err)
	}

	out := fixtures{Sources: []source{crabswarm, agents}, Issues: issues, Edges: edges}
	if err := writeFixtures(filepath.Join(dirs.web, "api", "fixtures.json"), out); err != nil {
		log.Fatalf("write fixtures: %v", err)
	}
	if err := writeRenderCSS(filepath.Join(dirs.web, "public", "assets", "css")); err != nil {
		log.Fatalf("write render css: %v", err)
	}
	log.Printf("wrote %d issues from %d sources to %s",
		len(out.Issues), len(out.Sources), filepath.Join(dirs.web, "api", "fixtures.json"))
}

// --- paths ------------------------------------------------------------------

type layout struct {
	mock   string // doc/plan/2026-09-04-plans_in_beads/mock
	plan   string // doc/plan/2026-09-04-plans_in_beads
	module string // repository/module root
	web    string // web/mock/plans_in_beads
}

// locate derives every path from this source file's own location, so the
// generator works from any working directory.
func locate() layout {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot locate gen.go")
	}
	mock := filepath.Dir(self)
	plan := filepath.Dir(mock)
	module := filepath.Dir(filepath.Dir(filepath.Dir(plan))) // <module>/doc/plan/<plan>
	return layout{
		mock:   mock,
		plan:   plan,
		module: module,
		web:    filepath.Join(module, "web", "mock", "plans_in_beads"),
	}
}

// --- input ------------------------------------------------------------------

// readExport reads the checked-in JSONL export next to this file. The
// `sample-title` row is a throwaway probe bead and is dropped.
func readExport(mockDir string) ([]exportRecord, error) {
	path := filepath.Join(mockDir, "issues-export.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checked-in export (refresh it with `bd export > %s`): %w", path, err)
	}

	var records []exportRecord
	for i, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec exportRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		if rec.Title == "sample-title" {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

// --- rendering --------------------------------------------------------------

type fieldRenderer struct {
	renderer *render.Renderer
}

// field renders one markdown text field. An empty field renders to an empty
// RenderedField, the way the proto describes it.
func (f *fieldRenderer) field(src string) renderedField {
	if strings.TrimSpace(src) == "" {
		return renderedField{TOC: []heading{}}
	}
	doc, err := f.renderer.Render([]byte(src))
	if err != nil {
		log.Fatalf("render markdown: %v", err)
	}
	toc := make([]heading, 0, len(doc.TOC))
	for _, h := range doc.TOC {
		toc = append(toc, heading{Level: h.Level, Text: h.Text, ID: h.ID})
	}
	return renderedField{HTML: string(doc.HTML), TOC: toc}
}

// --- backlog ----------------------------------------------------------------

func backlogIssues(r *fieldRenderer, srcID string, records []exportRecord) []issue {
	out := make([]issue, 0, len(records))
	for _, rec := range records {
		labels := rec.Labels
		if labels == nil {
			labels = []string{}
		}
		comments := make([]comment, 0, len(rec.Comments))
		for _, c := range rec.Comments {
			comments = append(comments, comment{
				ID:        c.ID,
				Author:    c.Author,
				Text:      r.field(c.Text),
				CreatedAt: c.CreatedAt,
			})
		}
		out = append(out, issue{
			SourceID: srcID,
			Summary: summary{
				ID:           rec.ID,
				Title:        rec.Title,
				IssueType:    rec.IssueType,
				Status:       protoStatus(rec.Status),
				Priority:     rec.Priority,
				Labels:       labels,
				CommentCount: rec.CommentCount,
				CreatedAt:    rec.CreatedAt,
				UpdatedAt:    rec.UpdatedAt,
				MetadataJSON: "{}",
			},
			Description:        r.field(rec.Description),
			Design:             r.field(rec.Design),
			AcceptanceCriteria: r.field(rec.AcceptanceCriteria),
			Notes:              r.field(rec.Notes),
			MetadataJSON:       "{}",
			CloseReason:        r.field(rec.CloseReason),
			Comments:           comments,
		})
	}
	return out
}

// --- edges ------------------------------------------------------------------

// planEdges are the plan issue's own graph: every step is a child of the plan,
// each step blocks the next (D1 / UC7), and the two discovered-from items
// point back at the plan (UC4).
func planEdges(srcID string, steps int) []edge {
	out := make([]edge, 0, 2*steps+len(discoveredFrom))
	for n := 1; n <= steps; n++ {
		id := fmt.Sprintf("%s.%d", planID, n)
		out = append(out, edge{SourceID: srcID, FromID: id, ToID: planID, Type: "parent-child"})
		if n > 1 {
			prev := fmt.Sprintf("%s.%d", planID, n-1)
			out = append(out, edge{SourceID: srcID, FromID: id, ToID: prev, Type: "blocks"})
		}
	}
	for _, id := range discoveredFrom {
		out = append(out, edge{SourceID: srcID, FromID: id, ToID: planID, Type: "discovered-from"})
	}
	return out
}

// backlogEdges are invented for the mock: the real database has no edges yet.
// They follow what the issue texts say — the admin TUI screen "depends on"
// the admin subcommand and the per-room history, its close reason spawned the
// WatchRoom follow-up, and the two completion items concern the same widget —
// so the graph has a `related` edge and a `blocks` chain outside the plan.
func backlogEdges(srcID string) []edge {
	mk := func(from, to, typ string) edge {
		return edge{SourceID: srcID, FromID: from, ToID: to, Type: typ}
	}
	return []edge{
		mk("crabswarm-125", "crabswarm-60t", "blocks"),
		mk("crabswarm-125", "crabswarm-hlt", "blocks"),
		mk("crabswarm-jp7", "crabswarm-125", "discovered-from"),
		mk("crabswarm-d46", "crabswarm-d7k", "related"),
	}
}

// agentsEdges hang the two open agents-package tasks under that source's
// invented plan issue, so the second source also has an epic lane.
func agentsEdges(srcID string) []edge {
	return []edge{
		{SourceID: srcID, FromID: "agents-package-k1x", ToID: agentsPlanID, Type: "parent-child"},
		{SourceID: srcID, FromID: "agents-package-m4d", ToID: agentsPlanID, Type: "parent-child"},
	}
}

// link derives from the edge list everything an issue carries about its
// relations — parentId, children with the counts behind the epic progress
// bar, and the dependencies table — so the fixture cannot contradict itself.
// parent-child edges are left out of the dependencies table: the children
// section and the parent link in the header already show them.
func link(issues []issue, edges []edge) error {
	byID := make(map[string]*issue, len(issues))
	for i := range issues {
		byID[issues[i].Summary.ID] = &issues[i]
		issues[i].Children = []summary{}
		issues[i].Depends = []dependency{}
	}
	for _, e := range edges {
		from, ok := byID[e.FromID]
		if !ok {
			return fmt.Errorf("edge %s -> %s: unknown issue %s", e.FromID, e.ToID, e.FromID)
		}
		to, ok := byID[e.ToID]
		if !ok {
			return fmt.Errorf("edge %s -> %s: unknown issue %s", e.FromID, e.ToID, e.ToID)
		}
		if e.Type == "parent-child" {
			from.Summary.ParentID = to.Summary.ID
			continue
		}
		from.Depends = append(from.Depends, dependency{ID: to.Summary.ID, Title: to.Summary.Title, Type: e.Type, Outgoing: true})
		to.Depends = append(to.Depends, dependency{ID: from.Summary.ID, Title: from.Summary.Title, Type: e.Type, Outgoing: false})
	}
	// Children in a second pass, once every ParentID is final.
	for i := range issues {
		p := issues[i].Summary.ParentID
		if p == "" {
			continue
		}
		parent := byID[p]
		parent.Children = append(parent.Children, issues[i].Summary)
		parent.Summary.ChildCount++
		if issues[i].Summary.Status == "ISSUE_STATUS_CLOSED" {
			parent.Summary.ChildClosedCount++
		}
	}
	return nil
}

// --- the plan issue ---------------------------------------------------------

// planIssues synthesizes the plan issue and its step children out of the plan
// directory's markdown, following D1's field convention. Relations (children,
// dependencies, counts) are filled in by link.
func planIssues(r *fieldRenderer, srcID, planDir string) (issue, []issue, error) {
	idea, err := os.ReadFile(filepath.Join(planDir, "IDEA.md"))
	if err != nil {
		return issue{}, nil, err
	}
	planMD, err := os.ReadFile(filepath.Join(planDir, "PLAN.md"))
	if err != nil {
		return issue{}, nil, err
	}
	decisions, err := os.ReadFile(filepath.Join(planDir, "DECISION.md"))
	if err != nil {
		return issue{}, nil, err
	}
	status, err := os.ReadFile(filepath.Join(planDir, "STATUS.md"))
	if err != nil {
		return issue{}, nil, err
	}

	acceptance := successCriteria(string(planMD))
	steps := stepIssues(r, srcID, string(planMD))
	comments := planComments(r, string(decisions))
	metadata := fmt.Sprintf("{%q:%q}", "idea_gate", ideaGate)

	plan := issue{
		SourceID: srcID,
		Summary: summary{
			ID:           planID,
			Title:        "Plans in beads",
			IssueType:    "epic",
			Status:       protoStatus("in_progress"),
			Priority:     1,
			Labels:       []string{"plan", "preview", "proto"},
			CommentCount: len(comments),
			CreatedAt:    "2026-09-04T09:12:00Z",
			UpdatedAt:    "2026-09-04T14:41:00Z",
			MetadataJSON: metadata,
		},
		Description:        r.field(string(idea)),
		Design:             r.field(string(planMD)),
		AcceptanceCriteria: r.field(acceptance),
		Notes:              r.field(string(status)),
		MetadataJSON:       metadata,
		CloseReason:        renderedField{TOC: []heading{}},
		Comments:           comments,
	}
	return plan, steps, nil
}

// successCriteria extracts PLAN.md's "Goal / success criteria" bullets, which
// D1 puts in the plan issue's acceptance_criteria.
func successCriteria(planMD string) string {
	body := section(planMD, "## Goal / success criteria")
	var keep []string
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "Derived from") {
			continue
		}
		keep = append(keep, line)
	}
	return strings.TrimSpace(strings.Join(keep, "\n")) + "\n"
}

// section returns the body of the `## <heading>` section of md, without the
// heading line and without the next section.
func section(md, heading string) string {
	lines := strings.Split(md, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

var decisionHeading = regexp.MustCompile(`(?m)^## (D\d+ .*)$`)

// planComments turns every DECISION.md entry into a `Decision:` comment and
// appends the two `Discussion:` comments this mock adds by hand. The prefix is
// what the GUI derives the comment badge from (D1: decisions and discussion
// are comments, distinguished by their first word).
func planComments(r *fieldRenderer, decisionsMD string) []comment {
	locs := decisionHeading.FindAllStringSubmatchIndex(decisionsMD, -1)
	out := make([]comment, 0, len(locs)+2)
	for i, loc := range locs {
		title := decisionsMD[loc[2]:loc[3]]
		bodyStart := loc[1]
		bodyEnd := len(decisionsMD)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := strings.TrimSpace(decisionsMD[bodyStart:bodyEnd])
		text := fmt.Sprintf("Decision: %s\n\n%s\n", title, body)
		// Seven minutes apart, in order: a decision log whose stamps run
		// backwards reads as a bug in the viewer.
		mins := i * 7
		out = append(out, comment{
			ID:        fmt.Sprintf("%s-comment-%02d", planID, i+1),
			Author:    "ngicks",
			Text:      r.field(text),
			CreatedAt: fmt.Sprintf("2026-09-04T%02d:%02d:00Z", 10+mins/60, mins%60),
		})
	}

	out = append(out, comment{
		ID:     fmt.Sprintf("%s-comment-%02d", planID, len(out)+1),
		Author: "claude",
		Text: r.field(`Discussion: naming — "issue" or "bead" in the GUI?

D12 settles the word: the CLI, the Go packages, the proto and the URLs all
say *issue*, and "beads" names only the store and its ` + "`bd`" + ` tool. This mock
follows that: the tab is **Issues**, the routes are ` + "`/issues/…`" + `, and the
only place the word "beads" appears is the source's ` + "`.beads`" + ` path in the
source switcher.

Open nit that this screen surfaces: a *plan* is an issue with
` + "`issue_type: epic`" + ` and the ` + "`plan`" + ` label, and nothing on the row says so
except the badges. The "plans only" filter is the workaround; a dedicated
plan badge would be a convention leaking into a generic viewer.
`),
		CreatedAt: "2026-09-04T13:20:00Z",
	})
	out = append(out, comment{
		ID:     fmt.Sprintf("%s-comment-%02d", planID, len(out)+1),
		Author: "claude",
		Text: r.field(`Discussion: freshness — what the reader sees when a step closes

D8 pushes ` + "`IssuesChanged`" + ` from the daemon's 10 s poll and the SPA
invalidates the affected queries, exactly as the file browser does for
` + "`WatchEvents`" + `. Two things are worth deciding while looking at this screen:

- Does an open issue view **re-render in place** (title, status badge and
  the rendered fields swap under the reader) or show a "changed — reload"
  affordance? In-place matches the file browser; it also means a diagram
  redraws while being read.
- The list is ordered newest-updated first, so a push can **reorder rows**
  under the cursor. The mock's "simulate change" button demonstrates both
  effects on one issue.
`),
		CreatedAt: "2026-09-04T13:44:00Z",
	})
	return out
}

var stepItem = regexp.MustCompile(`^(\d+)\. (.*)$`)
var stepTitle = regexp.MustCompile(`^\*\*(.+?)\*\*`)

// stepIssues turns PLAN.md's numbered implementation steps into child task
// issues; planEdges orders them by `blocks` dependencies (D1 / UC7).
func stepIssues(r *fieldRenderer, srcID, planMD string) []issue {
	type step struct {
		title string
		body  []string
	}
	var steps []step
	for line := range strings.SplitSeq(section(planMD, "## Implementation steps"), "\n") {
		if m := stepItem.FindStringSubmatch(line); m != nil {
			title := m[2]
			if t := stepTitle.FindStringSubmatch(m[2]); t != nil {
				title = strings.TrimSuffix(strings.ReplaceAll(t[1], "`", ""), ".")
			}
			steps = append(steps, step{title: title, body: []string{m[2]}})
			continue
		}
		if len(steps) == 0 {
			continue
		}
		cur := &steps[len(steps)-1]
		cur.body = append(cur.body, strings.TrimPrefix(line, "   "))
	}

	// Statuses per the mock's brief: 1-2 done, 3 running, the rest queued
	// (steps past the slice stay open).
	statuses := []string{"closed", "closed", "in_progress"}
	closeReasons := map[int]string{
		1: "Closed 2026-09-04. `Where` decodes bd's JSON envelope, `List` and `Get` decode\n" +
			"omitted fields as empty, and the fake-`bd` tests cover the `no_beads_directory`\n" +
			"error. Verified: `go test ./crabswarm/issues/...`.\n",
		2: "Closed 2026-09-04. One `mermaid-lint --format json --quiet` run over the temp\n" +
			"files maps back to issue, field, comment index and line. Verified:\n" +
			"`go test ./crabswarm/issues/mermaidlint/...` with the real CLI on PATH.\n",
	}

	out := make([]issue, 0, len(steps))
	for i, s := range steps {
		n := i + 1
		id := fmt.Sprintf("%s.%d", planID, n)
		status := "open"
		if i < len(statuses) {
			status = statuses[i]
		}
		updated := fmt.Sprintf("2026-09-04T%02d:00:00Z", 10+n)
		out = append(out, issue{
			SourceID: srcID,
			Summary: summary{
				ID:           id,
				Title:        fmt.Sprintf("Step %d — %s", n, s.title),
				IssueType:    "task",
				Status:       protoStatus(status),
				Priority:     priorityForStep(n),
				Labels:       []string{"step"},
				CreatedAt:    "2026-09-04T09:30:00Z",
				UpdatedAt:    updated,
				MetadataJSON: "{}",
			},
			Description: r.field(strings.TrimSpace(strings.Join(s.body, "\n")) + "\n"),
			Design:      renderedField{TOC: []heading{}},
			AcceptanceCriteria: r.field(fmt.Sprintf(
				"- The step's verification in PLAN.md passes.\n- `crabswarm issues lint` stays green on %s.\n",
				id,
			)),
			Notes:        renderedField{TOC: []heading{}},
			MetadataJSON: "{}",
			CloseReason:  r.field(closeReasons[n]),
			Comments:     []comment{},
		})
	}
	return out
}

func priorityForStep(n int) int {
	if n <= 3 {
		return 1
	}
	return 2
}

// --- the second, made-up source --------------------------------------------

// agentsIssues are four invented issues for a second registered source, so
// the source switcher has somewhere to switch to (D13: one source per
// repository): a plan issue with two open tasks under it, and one closed task.
func agentsIssues(r *fieldRenderer, srcID string) []issue {
	mk := func(id, title, typ, status string, priority int, labels []string, created, updated string) issue {
		return issue{
			SourceID: srcID,
			Summary: summary{
				ID: id, Title: title, IssueType: typ, Status: protoStatus(status),
				Priority: priority, Labels: labels, CreatedAt: created, UpdatedAt: updated,
				MetadataJSON: "{}",
			},
			Design:       renderedField{TOC: []heading{}},
			Notes:        renderedField{TOC: []heading{}},
			MetadataJSON: "{}",
			CloseReason:  renderedField{TOC: []heading{}},
			Comments:     []comment{},
		}
	}

	plan := mk(
		agentsPlanID,
		"ngplan authors plans in beads",
		"epic",
		"open",
		1,
		[]string{"plan", "ngplan"},
		"2026-09-04T15:00:00Z",
		"2026-09-04T15:41:00Z",
	)
	plan.Description = r.field(`# ngplan authors plans in beads — how it should be

Gate: not confirmed

The skill should create a plan issue with one ` + "`bd create`" + ` and keep the
gate, the decisions and the steps on that issue, following the convention
crabswarm's "plans in beads" plan fixes.

## Use cases

### UC1 — a fresh session resumes a plan from its issue

The agent reads the gate from ` + "`idea_gate`" + ` metadata, the decisions from the
comment thread and the next step from ` + "`bd ready`" + `.
`)
	plan.AcceptanceCriteria = r.field("- The skill never writes `doc/plan/` files.\n- Every plan it makes renders in crabswarm preview.\n")

	skill := mk(
		"agents-package-k1x",
		"Rewrite the ngplan skill to author plan issues",
		"task",
		"blocked",
		1,
		[]string{"ngplan", "skill"},
		"2026-09-04T15:02:00Z",
		"2026-09-04T15:40:00Z",
	)
	skill.Description = r.field(`Recorded 2026-09-04, from crabswarm's "plans in beads" plan.

The skill writes ` + "`doc/plan/<date>-<slug>/`" + ` files today. It should create one
plan issue instead: idea in ` + "`description`" + `, plan in ` + "`design`" + `, criteria in
` + "`acceptance_criteria`" + `, narrative in ` + "`notes`" + `, decisions as
` + "`Decision:`" + ` comments, steps as child tasks.

` + "```mermaid" + `
flowchart LR
    skill[ngplan skill] -->|bd create| plan[plan issue]
    plan -->|children| steps[step tasks]
    plan -->|comments| log[decision log]
` + "```" + `

Blocked until crabswarm's convention is proven against its own GUI (D4).
`)
	skill.AcceptanceCriteria = r.field(
		"- One `bd create` produces a plan issue a fresh session can read.\n" +
			"- The gate is written as `idea_gate` metadata, not chat memory.\n",
	)

	hook := mk("agents-package-m4d", "Package the issues mermaid Stop hook", "task", "open", 2,
		[]string{"apm", "hooks", "mermaid"}, "2026-09-04T15:05:00Z", "2026-09-04T15:05:00Z")
	hook.Description = r.field(
		"Recorded 2026-09-04.\n\nD11 keeps `hooks/issues-mermaid-lint` in crabswarm for now. Once the plan\n" +
			"convention settles, move the package here beside `hooks/markdown-mermaid-lint`\n" +
			"so both guards ship from one place.\n",
	)

	docs := mk(
		"agents-package-7ry",
		"Document the discovered-from handoff flow",
		"task",
		"closed",
		2,
		[]string{"docs", "handoff"},
		"2026-09-03T11:00:00Z",
		"2026-09-04T08:12:00Z",
	)
	docs.Description = r.field(
		"Recorded 2026-09-03.\n\nThe handoff section still describes a `HANDOFF.md` fold step. Replace it with\n" +
			"the `discovered-from` dependency (UC4): the item is born in the backlog.\n",
	)
	docs.CloseReason = r.field(
		"Closed 2026-09-04. The instructions now describe `bd create` with\n" +
			"`--dep discovered-from:<step id>` and no fold moment.\n",
	)

	return []issue{plan, skill, hook, docs}
}

// --- output -----------------------------------------------------------------

// sourceID mirrors "a stable identifier derived from the absolute .beads path".
func sourceID(beadsPath string) string {
	sum := sha256.Sum256([]byte(beadsPath))
	return hex.EncodeToString(sum[:])[:12]
}

// protoStatus maps bd's status string onto the IssueStatus enum name, the
// spelling protobuf-JSON uses on the wire.
func protoStatus(s string) string {
	switch s {
	case "open":
		return "ISSUE_STATUS_OPEN"
	case "in_progress":
		return "ISSUE_STATUS_IN_PROGRESS"
	case "blocked":
		return "ISSUE_STATUS_BLOCKED"
	case "closed":
		return "ISSUE_STATUS_CLOSED"
	default:
		return "ISSUE_STATUS_UNSPECIFIED"
	}
}

// writeFixtures writes the fixture file with HTML escaping off, so the
// rendered markup stays readable (and greppable) in the JSON.
func writeFixtures(path string, out fixtures) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	return f.Close()
}

// writeRenderCSS drops the stylesheets the rendered HTML expects into the
// mock's public/ directory. The real previewer serves them from the daemon
// (httpapi.AlertCSSPath, ChromaLightCSSPath, ChromaDarkCSSPath); the mock has
// no daemon, so signals/ui.ts's <link> tags would 404 without these files.
func writeRenderCSS(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	sheets, err := render.ChromaStylesheets()
	if err != nil {
		return err
	}
	files := map[string]string{"alert.css": alert.CSS}
	for name, css := range sheets {
		files[fmt.Sprintf("chroma-%s.css", name)] = css
	}
	for name, css := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(css), 0o644); err != nil {
			return err
		}
	}
	return nil
}
