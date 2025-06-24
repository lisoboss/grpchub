package header

import (
	"strings"
)

type Header map[string][]string

func (h Header) Append(k string, vals ...string) {
	if len(vals) == 0 {
		return
	}
	k = strings.ToLower(k)
	h[k] = append(h[k], vals...)
}

func (h Header) Extend(header Header) {
	for key, vals := range header {
		h.Append(key, vals...)
	}
}
