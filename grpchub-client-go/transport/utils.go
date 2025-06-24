package transport

import (
	"io"

	"github.com/lisoboss/grpchub/grpchublog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = grpchublog.Component("grpchub-transport")

func checkIoError(err error) error {
	st, ok := status.FromError(err)
	if ok && st.Code() == codes.Canceled {
		// logger.Warningf("tunnel recv canceled: %v", st.Message())
		return io.EOF
	}
	return err
}
