package grpcx

import (
	"fmt"
	"strings"
)

func parseFullMethod(sm string) (string, string, error) {
	if sm != "" && sm[0] == '/' {
		sm = sm[1:]
	}
	pos := strings.LastIndex(sm, "/")

	if pos == -1 {
		return "", "", fmt.Errorf("malformed method name: %q", sm)
	}

	return sm[:pos], sm[pos+1:], nil
}
