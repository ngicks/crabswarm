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
// notes, decisions as comments, implementation steps as child tasks), renders
// every markdown field through the previewer's own renderer
// (crabswarm/preview/render) and writes:
//
//	web/mock/plans_in_beads/fixtures.json                    the mock's data
//	web/mock/plans_in_beads/public/assets/css/alert.css      served at the
//	web/mock/plans_in_beads/public/assets/css/chroma-*.css   paths signals/ui.ts links
//
// fixtures.json is shaped like the messages of PLAN.md's proto sketch
// (ngicks.crabswarm.issues.v1: Source, IssueSummary, RenderedField,
// IssueComment, IssueDependency, Issue) in protobuf-JSON spelling: camelCase
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

// planID is the ID the dogfood plan issue (PLAN.md step 8) would get.
const planID = "crabswarm-plan1"

// ideaGate is D7's metadata value: the date the idea gate was confirmed.
const ideaGate = "2026-09-04"

// discoveredFrom are two real backlog issues presented as born from this plan
// (D1 / UC4). Both are referenced by the plan text itself.
var discoveredFrom = []string{"crabswarm-d2j", "crabswarm-aki"}

// --- fixture shapes (protobuf-JSON spelling of PLAN.md's issues/v1) ---------

type fixtures struct {
	Sources []source `json:"sources"`
	Issues  []issue  `json:"issues"`
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
}

// comment mirrors issues.v1.IssueComment.
type comment struct {
	ID        string        `json:"id"`
	Author    string        `json:"author"`
	Text      renderedField `json:"text"`
	CreatedAt string        `json:"createdAt"`
}

// dependency mirrors issues.v1.IssueDependency.
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
	plan, steps, err := planIssues(r, crabswarm.ID, dirs.plan, backlog)
	if err != nil {
		log.Fatalf("build plan issue: %v", err)
	}
	linkDiscoveredFrom(backlog, plan.Summary)

	issues := make([]issue, 0, len(backlog)+len(steps)+4)
	issues = append(issues, plan)
	issues = append(issues, steps...)
	issues = append(issues, backlog...)
	issues = append(issues, agentsIssues(r, agents.ID)...)

	out := fixtures{Sources: []source{crabswarm, agents}, Issues: issues}
	if err := writeFixtures(filepath.Join(dirs.web, "fixtures.json"), out); err != nil {
		log.Fatalf("write fixtures: %v", err)
	}
	if err := writeRenderCSS(filepath.Join(dirs.web, "public", "assets", "css")); err != nil {
		log.Fatalf("write render css: %v", err)
	}
	log.Printf("wrote %d issues from %d sources to %s",
		len(out.Issues), len(out.Sources), filepath.Join(dirs.web, "fixtures.json"))
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
			},
			Description:        r.field(rec.Description),
			Design:             r.field(rec.Design),
			AcceptanceCriteria: r.field(rec.AcceptanceCriteria),
			Notes:              r.field(rec.Notes),
			MetadataJSON:       "{}",
			CloseReason:        r.field(rec.CloseReason),
			Comments:           comments,
			Children:           []summary{},
			Depends:            []dependency{},
		})
	}
	return out
}

// linkDiscoveredFrom marks the two chosen backlog issues as discovered from
// the plan (UC4: a handoff item is born linked to its origin).
func linkDiscoveredFrom(backlog []issue, plan summary) {
	for _, id := range discoveredFrom {
		for i := range backlog {
			if backlog[i].Summary.ID != id {
				continue
			}
			backlog[i].Depends = append(backlog[i].Depends, dependency{
				ID:       plan.ID,
				Title:    plan.Title,
				Type:     "discovered-from",
				Outgoing: true,
			})
		}
	}
}

// --- the plan issue ---------------------------------------------------------

// planIssues synthesizes the plan issue and its eight step children out of the
// plan directory's markdown, following D1's field convention.
func planIssues(r *fieldRenderer, srcID, planDir string, backlog []issue) (issue, []issue, error) {
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

	children := make([]summary, 0, len(steps))
	for _, s := range steps {
		children = append(children, s.Summary)
	}

	deps := make([]dependency, 0, len(discoveredFrom))
	for _, id := range discoveredFrom {
		title := id
		for _, b := range backlog {
			if b.Summary.ID == id {
				title = b.Summary.Title
			}
		}
		deps = append(
			deps,
			dependency{ID: id, Title: title, Type: "discovered-from", Outgoing: false},
		)
	}

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
			ChildCount:   len(children),
			CreatedAt:    "2026-09-04T09:12:00Z",
			UpdatedAt:    "2026-09-04T14:41:00Z",
		},
		Description:        r.field(string(idea)),
		Design:             r.field(string(planMD)),
		AcceptanceCriteria: r.field(acceptance),
		Notes:              r.field(string(status)),
		MetadataJSON:       fmt.Sprintf("{%q:%q}", "idea_gate", ideaGate),
		CloseReason:        renderedField{TOC: []heading{}},
		Comments:           comments,
		Children:           children,
		Depends:            deps,
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
// issues, ordered by `blocks` dependencies (D1 / UC7).
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

	// Statuses per the mock's brief: 1-2 done, 3 running, the rest queued.
	statuses := []string{"closed", "closed", "in_progress", "open", "open", "open", "open", "open"}
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
		deps := make([]dependency, 0, 2)
		if n > 1 {
			deps = append(deps, dependency{
				ID:       fmt.Sprintf("%s.%d", planID, n-1),
				Title:    fmt.Sprintf("Step %d — %s", n-1, steps[i-1].title),
				Type:     "blocks",
				Outgoing: false,
			})
		}
		if n < len(steps) {
			deps = append(deps, dependency{
				ID:       fmt.Sprintf("%s.%d", planID, n+1),
				Title:    fmt.Sprintf("Step %d — %s", n+1, steps[i+1].title),
				Type:     "blocks",
				Outgoing: true,
			})
		}
		updated := fmt.Sprintf("2026-09-04T%02d:00:00Z", 10+n)
		out = append(out, issue{
			SourceID: srcID,
			Summary: summary{
				ID:        id,
				Title:     fmt.Sprintf("Step %d — %s", n, s.title),
				IssueType: "task",
				Status:    protoStatus(status),
				Priority:  priorityForStep(n),
				Labels:    []string{"step"},
				ParentID:  planID,
				CreatedAt: "2026-09-04T09:30:00Z",
				UpdatedAt: updated,
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
			Children:     []summary{},
			Depends:      deps,
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

// agentsIssues are three invented issues for a second registered source, so
// the source switcher has somewhere to switch to (D13: one source per
// repository).
func agentsIssues(r *fieldRenderer, srcID string) []issue {
	mk := func(id, title, typ, status string, priority int, labels []string, created, updated string) issue {
		return issue{
			SourceID: srcID,
			Summary: summary{
				ID: id, Title: title, IssueType: typ, Status: protoStatus(status),
				Priority: priority, Labels: labels, CreatedAt: created, UpdatedAt: updated,
			},
			Design:       renderedField{TOC: []heading{}},
			Notes:        renderedField{TOC: []heading{}},
			MetadataJSON: "{}",
			CloseReason:  renderedField{TOC: []heading{}},
			Comments:     []comment{},
			Children:     []summary{},
			Depends:      []dependency{},
		}
	}

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

	return []issue{skill, hook, docs}
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
