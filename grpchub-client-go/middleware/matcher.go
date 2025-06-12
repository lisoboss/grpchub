package middleware

import (
	"sort"
	"strings"
)

// NewMatcher new a middleware matcher.
func NewMatcher() *Matcher {
	return &Matcher{
		matchs: make(map[string][]Middleware),
	}
}

type Matcher struct {
	prefix   []string
	defaults []Middleware
	matchs   map[string][]Middleware
}

func (m *Matcher) Use(ms ...Middleware) {
	m.defaults = ms
}

func (m *Matcher) Add(selector string, ms ...Middleware) {
	if strings.HasSuffix(selector, "*") {
		selector = strings.TrimSuffix(selector, "*")
		m.prefix = append(m.prefix, selector)
		// sort the prefix:
		//  - /foo/bar
		//  - /foo
		sort.Slice(m.prefix, func(i, j int) bool {
			return m.prefix[i] > m.prefix[j]
		})
	}
	m.matchs[selector] = ms
}

func (m *Matcher) Match(operation string) []Middleware {
	ms := make([]Middleware, 0, len(m.defaults))
	if len(m.defaults) > 0 {
		ms = append(ms, m.defaults...)
	}
	if next, ok := m.matchs[operation]; ok {
		return append(ms, next...)
	}
	for _, prefix := range m.prefix {
		if strings.HasPrefix(operation, prefix) {
			return append(ms, m.matchs[prefix]...)
		}
	}
	return ms
}
