package preview

import (
	"net"
	"net/http"

	"github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1/previewv1connect"
)

// NewClient returns a connect-go client for the preview daemon that listens at
// addr. addr is the daemon's TCP *listen* address (e.g. the configured
// "0.0.0.0:6419"); it is turned into a *dialable* base URL by [baseURL]. The
// transport is plain HTTP — the previewer serves cleartext and relies on the
// tailnet/LAN boundary for confidentiality, so the local CLI does not attempt
// TLS.
func NewClient(addr string) previewv1connect.PreviewServiceClient {
	return previewv1connect.NewPreviewServiceClient(http.DefaultClient, baseURL(addr))
}

// baseURL turns a listen address into the base URL the local CLI dials, e.g.
// "0.0.0.0:6419" -> "http://127.0.0.1:6419". It is shared by [NewClient] and
// the /healthz poll in EnsureDaemon so both reach the daemon the same way.
func baseURL(addr string) string {
	return "http://" + dialableHostPort(addr)
}

// dialableHostPort rewrites a wildcard listen host to a loopback address so the
// local CLI can actually connect. A server bound to "0.0.0.0" (or ":port", the
// empty host) or the IPv6 wildcard "::" accepts connections on every interface,
// but those wildcard addresses are not themselves routable as *destinations* —
// dialing "0.0.0.0:6419" fails. The reachable local equivalent is the loopback
// interface, so wildcard hosts are mapped to 127.0.0.1 / ::1. Any concrete host
// (a hostname, a specific IP, a Tailscale MagicDNS name) is left untouched.
//
// If addr carries no port or is otherwise unparsable, it is returned verbatim;
// callers pass the configured address, so a malformed one surfaces later as a
// connection error rather than being silently rewritten here.
func dialableHostPort(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	switch host {
	case "", "0.0.0.0":
		host = "127.0.0.1"
	case "::":
		host = "::1"
	}
	return net.JoinHostPort(host, port)
}
