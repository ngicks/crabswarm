package git

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// ParseRepoPath derives a ghq-style relative repository path
// ("host/owner/repo") from a clone URL. The result uses forward slashes
// regardless of OS; callers join it under the base directory with
// [path/filepath.FromSlash] + [path/filepath.Join].
//
// Recognized forms (with or without a trailing ".git"):
//
//	https://github.com/owner/repo.git      → github.com/owner/repo
//	http://example.com/owner/repo          → example.com/owner/repo
//	ssh://git@github.com/owner/repo.git    → github.com/owner/repo
//	git@github.com:owner/repo.git          → github.com/owner/repo   (scp-like)
//	github.com/owner/repo                  → github.com/owner/repo   (scheme-less)
//
// Any userinfo and port are dropped from the host. The path keeps its full
// depth, so nested namespaces (GitLab subgroups, Gerrit, etc.) survive.
func ParseRepoPath(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("empty repository URL")
	}

	var host, p string
	switch {
	case strings.Contains(s, "://"):
		u, err := url.Parse(s)
		if err != nil {
			return "", fmt.Errorf("parsing url %q: %w", raw, err)
		}
		host, p = u.Hostname(), u.Path
	case isSCPLike(s):
		hostPart, pathPart, _ := strings.Cut(s, ":")
		if at := strings.LastIndex(hostPart, "@"); at >= 0 {
			hostPart = hostPart[at+1:]
		}
		host, p = hostPart, pathPart
	default:
		// Scheme-less "host/owner/repo".
		host, p, _ = strings.Cut(s, "/")
	}

	host = strings.TrimSpace(host)
	p = strings.Trim(strings.TrimSpace(p), "/")
	p = strings.TrimSuffix(p, ".git")
	p = strings.Trim(p, "/")

	if host == "" || p == "" {
		return "", fmt.Errorf("cannot derive host/path from %q", raw)
	}

	// Collapse any "." / ".." and stray slashes; reject escapes outside host.
	rel := path.Clean(host + "/" + p)
	if rel == "." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
		return "", fmt.Errorf("invalid repository path derived from %q", raw)
	}
	return rel, nil
}

// isSCPLike reports whether s uses git's scp-like syntax
// ("[user@]host:path") rather than a URL or a scheme-less path. The
// distinguishing mark is a colon that appears before the first slash and is
// not part of a "://" scheme separator.
func isSCPLike(s string) bool {
	if strings.Contains(s, "://") {
		return false
	}
	colon := strings.IndexByte(s, ':')
	if colon < 0 {
		return false
	}
	slash := strings.IndexByte(s, '/')
	return slash < 0 || colon < slash
}
