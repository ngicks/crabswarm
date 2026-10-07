package poll

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
)

// HTTPOption shapes the request a KindHTTP probe sends and the statuses that
// count as ready. The zero value sends a bodiless GET and accepts any 2xx.
type HTTPOption struct {
	// Method is the request method. Empty selects GET.
	Method string
	// Header is added to every probe. A "Host" entry sets the request's Host.
	Header http.Header
	// Body is sent with every probe. Nil sends none.
	Body []byte
	// Status lists the statuses that count as ready. Empty accepts any 2xx.
	Status []StatusRange
}

// IsZero reports whether o asks for nothing beyond the default probe, which is
// what a target other than KindHTTP accepts.
func (o HTTPOption) IsZero() bool {
	return o.Method == "" && len(o.Header) == 0 && o.Body == nil && len(o.Status) == 0
}

// StatusRange is an inclusive range of HTTP status codes.
type StatusRange struct {
	Min, Max int
}

// ParseStatus reads status specs, each a comma-separated list of codes
// ("204") and classes ("2xx").
func ParseStatus(specs []string) ([]StatusRange, error) {
	var out []StatusRange
	for _, spec := range specs {
		for item := range strings.SplitSeq(spec, ",") {
			item = strings.TrimSpace(item)
			r, err := parseStatusItem(item)
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
	}
	return out, nil
}

func parseStatusItem(item string) (StatusRange, error) {
	if len(item) == 3 && strings.EqualFold(item[1:], "xx") && item[0] >= '1' && item[0] <= '5' {
		base := int(item[0]-'0') * 100
		return StatusRange{Min: base, Max: base + 99}, nil
	}
	code, err := strconv.Atoi(item)
	if err != nil || code < 100 || code > 599 {
		return StatusRange{}, fmt.Errorf(
			"status %q is neither a code from 100 to 599 nor a class like 2xx", item)
	}
	return StatusRange{Min: code, Max: code}, nil
}

// ParseHeader reads "Name: value" lines into a header. A name repeated across
// lines keeps every value.
func ParseHeader(lines []string) (http.Header, error) {
	h := http.Header{}
	for _, line := range lines {
		name, value, ok := strings.Cut(line, ":")
		name = strings.TrimSpace(name)
		if !ok || name == "" || strings.ContainsAny(name, " \t") {
			return nil, fmt.Errorf("header %q is not in the form \"Name: value\"", line)
		}
		h.Add(textproto.CanonicalMIMEHeaderKey(name), strings.TrimSpace(value))
	}
	return h, nil
}

// accepts reports whether code counts as ready under o.
func (o HTTPOption) accepts(code int) bool {
	if len(o.Status) == 0 {
		return code >= 200 && code <= 299
	}
	for _, r := range o.Status {
		if code >= r.Min && code <= r.Max {
			return true
		}
	}
	return false
}

// probeHTTP reports whether the URL answers the request o describes with a
// status o accepts.
func probeHTTP(ctx context.Context, client *http.Client, url string, o HTTPOption) error {
	method := o.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if o.Body != nil {
		body = bytes.NewReader(o.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	for name, values := range o.Header {
		for _, v := range values {
			req.Header.Add(name, v)
		}
	}
	// net/http writes the Host line from req.Host and ignores a Host entry in
	// req.Header.
	if host := o.Header.Get("Host"); host != "" {
		req.Host = host
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Drain the body so the connection goes back to the pool for the next probe.
	_, _ = io.Copy(io.Discard, resp.Body)

	if !o.accepts(resp.StatusCode) {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return nil
}
