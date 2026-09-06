package issues

import (
	"context"
	"encoding/json"
	"fmt"
)

// ListFull returns the issues matching f as whole records. `bd list --json`
// reports every field of an issue, its metadata object included, so a caller
// that needs more than a [Summary] carries — the metadata chips of a list
// view, say — gets it from the same call instead of one `bd show` per listed
// issue.
func (c *Client) ListFull(ctx context.Context, f ListFilter) ([]Issue, error) {
	out, err := c.run(ctx, append([]string{"list", "--json"}, f.args()...)...)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("decoding bd list: %w", err)
	}
	return issues, nil
}
