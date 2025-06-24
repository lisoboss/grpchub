package grpcx

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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

type Scan anypb.Any

func (s *Scan) ScanAnyPb(reply any) error {
	m, ok := reply.(proto.Message)
	if !ok {
		return fmt.Errorf("")
	}
	return anypb.UnmarshalTo((*anypb.Any)(s), m, proto.UnmarshalOptions{})
}
