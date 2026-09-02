package templateutil

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

// This file holds the path-component and list helpers: splitting a path
// into components, keeping the tail of a list, and joining it back.

// SplitPath returns the components of filepath.Clean(path) split on the OS
// path separator. An absolute path yields a leading empty component (so
// `join "/"` of the result reproduces the path), a relative path does not,
// and "" yields nil. Pair it with LastN to keep only the trailing components
// of a long directory, e.g.
//
//	{{ splitPath .Workspace.CurrentDir | lastN 3 | join "/" }}
func SplitPath(path string) []string {
	if path == "" {
		return nil
	}
	return strings.Split(filepath.Clean(path), string(filepath.Separator))
}

// LastN returns the last n elements of list, which must be a slice or an
// array; the result is a slice of the same element type. When n is at least
// the length every element is returned, and n <= 0 yields an empty slice. A
// nil list yields nil. Any other kind of list is an error, so a template
// applying it to a non-list fails to render instead of printing garbage.
func LastN(n int, list any) (any, error) {
	if list == nil {
		return nil, nil
	}
	v := reflect.ValueOf(list)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return nil, fmt.Errorf("lastN: want a slice or array, got %s", v.Type())
	}
	length := v.Len()
	start := max(0, length-max(0, n))
	if v.Kind() == reflect.Array {
		// Slicing an array requires it to be addressable; copy it first.
		tmp := reflect.New(v.Type()).Elem()
		tmp.Set(v)
		v = tmp
	}
	return v.Slice(start, length).Interface(), nil
}

// Join concatenates the elements of list, which must be a slice or an array,
// with sep between them, like [strings.Join] with the arguments swapped so
// the list can be piped in: {{ splitPath .Cwd | lastN 3 | join "/" }}.
// Non-string elements are formatted with [fmt.Sprint]. A nil list yields "".
// Any other kind of list is an error, so a template applying it to a non-list
// fails to render instead of printing garbage.
func Join(sep string, list any) (string, error) {
	if list == nil {
		return "", nil
	}
	if ss, ok := list.([]string); ok {
		return strings.Join(ss, sep), nil
	}
	v := reflect.ValueOf(list)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return "", fmt.Errorf("join: want a slice or array, got %s", v.Type())
	}
	var b strings.Builder
	for i := range v.Len() {
		if i > 0 {
			b.WriteString(sep)
		}
		fmt.Fprint(&b, v.Index(i).Interface())
	}
	return b.String(), nil
}
