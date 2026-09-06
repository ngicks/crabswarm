package httpapi_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"gotest.tools/v3/assert"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
	"github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1/issuesv1connect"
)

// TestIssuesServiceMounted checks the previewer answers the issues API on the
// same server as the preview API, so the SPA reaches both over one origin.
func TestIssuesServiceMounted(t *testing.T) {
	_, base := startService(t)
	client := issuesv1connect.NewIssuesServiceClient(http.DefaultClient, base)

	res, err := client.ListSources(t.Context(),
		connect.NewRequest(&issuesv1.ListSourcesRequest{}))
	assert.NilError(t, err)
	assert.Equal(t, len(res.Msg.GetSources()), 0)

	// The dependency graph is a later step: a mounted route answers
	// Unimplemented, an unmounted one would 404.
	_, err = client.ListDependencies(t.Context(),
		connect.NewRequest(&issuesv1.ListDependenciesRequest{}))
	assert.Equal(t, connect.CodeOf(err), connect.CodeUnimplemented)
}
