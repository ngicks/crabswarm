package cli

import (
	"fmt"
	"io"
	"net"
	"os"
	"text/tabwriter"

	"connectrpc.com/connect"
)

// PreviewRoot is the presentation view of a registered preview root: the three
// columns `preview list` prints. It decouples the CLI table renderer from the
// generated protobuf type so this package stays independent of the wire schema.
type PreviewRoot struct {
	ID   string
	Name string
	Path string
}

// RenderRoots writes the registered preview roots to w as an aligned three-column
// table (ID, NAME, PATH). The header is always printed so the output is stable
// and greppable even when no roots are registered.
func RenderRoots(w io.Writer, roots []PreviewRoot) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tPATH")
	for _, r := range roots {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", r.ID, r.Name, r.Path)
	}
	return tw.Flush()
}

// PreviewURL builds the browser URL that opens the root rootID: the previewer
// serves each root under /r/<rootID>/. addr is the server's TCP *listen*
// address; when its host is a wildcard ("", "0.0.0.0" or "::") the server binds
// every interface but that host is not a routable destination, so the URL uses
// this machine's hostname (falling back to "localhost") — a name a phone or
// tablet on the tailnet can actually open, unlike the loopback the local CLI
// dials. A concrete host (a name, an IP, a MagicDNS address) is kept as-is.
func PreviewURL(addr, rootID string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// addr is not host:port shaped; use it verbatim rather than guessing.
		return fmt.Sprintf("http://%s/r/%s/", addr, rootID)
	}
	switch host {
	case "", "0.0.0.0", "::":
		host = displayHost()
	}
	return fmt.Sprintf("http://%s/r/%s/", net.JoinHostPort(host, port), rootID)
}

// displayHost returns this machine's hostname for a user-facing preview URL,
// falling back to "localhost" when the hostname cannot be determined.
func displayHost() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "localhost"
}

// PreviewDaemonError augments a preview RPC error with a start-the-daemon hint
// when the failure is the daemon being unreachable, and returns any other error
// unchanged. A refused connection surfaces from connect-go as CodeUnavailable
// (transport errors that are not already connect errors are wrapped with that
// code), which distinguishes "daemon not running" from an application error such
// as a not-found root. daemonName names the cmdman command to inspect.
func PreviewDaemonError(err error, daemonName string) error {
	if err == nil {
		return nil
	}
	if connect.CodeOf(err) == connect.CodeUnavailable {
		return fmt.Errorf(
			"preview daemon unreachable: %w\n"+
				"hint: start it by running `crabswarm preview` (which launches the daemon), "+
				"or inspect it with `cmdman inspect %s`",
			err, daemonName,
		)
	}
	return err
}
