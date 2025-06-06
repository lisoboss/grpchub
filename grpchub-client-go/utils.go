package grpchub

import (
	"fmt"

	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
	"github.com/lisoboss/grpchub/grpchublog"
	"google.golang.org/grpc/metadata"
)

var logger = grpchublog.Component("grpchub")

func parseMetadataEntriesTo(entries []*channelv1.MetadataEntry, reply any) error {
	md, ok := reply.(*metadata.MD)
	if !ok {
		return fmt.Errorf("reply is not *metadata.MD: got %T", reply)
	}
	for _, e := range entries {
		md.Append(e.Key, e.Values...)
	}
	return nil
}

func parseMetadataEntries(entries []*channelv1.MetadataEntry) metadata.MD {
	md := metadata.MD{}
	parseMetadataEntriesTo(entries, &md)
	return md
}

func buildMetadataEntries(md metadata.MD) []*channelv1.MetadataEntry {
	var entries []*channelv1.MetadataEntry
	for k, vals := range md {
		entries = append(entries, &channelv1.MetadataEntry{
			Key:    k,
			Values: vals,
		})
	}
	return entries
}
