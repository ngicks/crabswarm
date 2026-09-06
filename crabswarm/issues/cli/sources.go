// Package cli renders what the issues commands print and resolves what the
// user typed to the thing it names.
package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// RegistrationKind tells apart the two kinds of thing the preview daemon
// registers.
type RegistrationKind string

const (
	// KindRoot is a file root: a directory the previewer serves documents from.
	KindRoot RegistrationKind = "root"
	// KindSource is an issue source: a beads database the previewer reads
	// issues from.
	KindSource RegistrationKind = "source"
)

// Registration is the presentation view of one entry registered with the
// preview daemon — the four columns `preview list` prints. It decouples the
// table renderer and the lookup below from the generated protobuf types so
// this package stays independent of the wire schema.
type Registration struct {
	// Kind is which registry the entry lives in.
	Kind RegistrationKind
	// ID is the identifier the daemon assigned.
	ID string
	// Name is the display name: a root's name, a source's issue-ID prefix.
	Name string
	// Path is a root's directory, or a source's .beads directory.
	Path string
}

// RenderRegistrations writes the daemon's registered roots and issue sources
// to w as an aligned four-column table (KIND, ID, NAME, PATH). The header is
// always printed so the output is stable and greppable even when nothing is
// registered.
func RenderRegistrations(w io.Writer, regs []Registration) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "KIND\tID\tNAME\tPATH")
	for _, r := range regs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Kind, r.ID, r.Name, r.Path)
	}
	return tw.Flush()
}

// ResolveRegistration returns the single registration idOrName names, matching
// it against both ID and Name across roots and sources alike.
//
// Nothing guarantees the argument picks out one entry: two beads databases can
// share an issue-ID prefix, and a prefix can equal a root's name. Several
// matches is therefore reported as an error naming them, rather than removing
// whichever came first.
func ResolveRegistration(regs []Registration, idOrName string) (Registration, error) {
	var matched []Registration
	for _, r := range regs {
		if r.ID == idOrName || r.Name == idOrName {
			matched = append(matched, r)
		}
	}
	switch len(matched) {
	case 0:
		return Registration{}, fmt.Errorf("no root or issue source named %q", idOrName)
	case 1:
		return matched[0], nil
	default:
		return Registration{}, fmt.Errorf("%q names %d registrations (%s); use an ID",
			idOrName, len(matched), describeRegistrations(matched))
	}
}

// describeRegistrations lists registrations as "<kind> <id>", comma-separated,
// for the ambiguity error.
func describeRegistrations(regs []Registration) string {
	parts := make([]string, len(regs))
	for i, r := range regs {
		parts[i] = fmt.Sprintf("%s %s", r.Kind, r.ID)
	}
	return strings.Join(parts, ", ")
}
