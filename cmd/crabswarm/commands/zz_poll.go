package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/pkg/util/poll"
)

// pollHTTPFlags are the flags that shape an HTTP readiness probe. `util poll
// wait` registers them bare and `util supervised` under a "poll-" prefix.
type pollHTTPFlags struct {
	prefix  string
	method  string
	status  []string
	header  []string
	payload string
	file    string
}

func (f *pollHTTPFlags) register(cmd *cobra.Command, prefix string) {
	f.prefix = prefix
	fs := cmd.Flags()
	fs.StringVar(&f.method, prefix+"method", "",
		"HTTP method of the probe (default GET). HEAD is the cheapest; "+
			"use GET for an endpoint that mishandles HEAD. Any other method, e.g. QUERY, "+
			"is sent upper-cased")
	fs.StringSliceVar(&f.status, prefix+"status", nil,
		"HTTP statuses that count as ready: codes (204) and classes (2xx), "+
			"comma-separated or repeated (default 2xx)")
	fs.StringArrayVar(&f.header, prefix+"header", nil,
		`header added to every HTTP probe as "Name: value"; repeatable`)
	fs.StringVar(&f.payload, prefix+"payload", "",
		"request body sent with every HTTP probe")
	fs.StringVar(&f.file, prefix+"file", "",
		`file whose content is the request body of every HTTP probe ("-" reads stdin)`)
	cmd.MarkFlagsMutuallyExclusive(prefix+"payload", prefix+"file")
}

// option resolves the flags into the probe they describe. stdin is what a
// --file of "-" reads.
func (f *pollHTTPFlags) option(cmd *cobra.Command, stdin io.Reader) (poll.HTTPOption, error) {
	var opt poll.HTTPOption
	opt.Method = strings.ToUpper(f.method)

	status, err := poll.ParseStatus(f.status)
	if err != nil {
		return opt, fmt.Errorf("--%sstatus: %w", f.prefix, err)
	}
	opt.Status = status

	if len(f.header) > 0 {
		if opt.Header, err = poll.ParseHeader(f.header); err != nil {
			return opt, fmt.Errorf("--%sheader: %w", f.prefix, err)
		}
	}

	switch {
	case cmd.Flags().Changed(f.prefix + "payload"):
		opt.Body = []byte(f.payload)
	case f.file == "-":
		if opt.Body, err = io.ReadAll(stdin); err != nil {
			return opt, fmt.Errorf("--%sfile: reading stdin: %w", f.prefix, err)
		}
	case f.file != "":
		if opt.Body, err = os.ReadFile(f.file); err != nil {
			return opt, fmt.Errorf("--%sfile: %w", f.prefix, err)
		}
	}
	return opt, nil
}
