package grpchub

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	channelv1 "github.com/lisoboss/grpchub/gen/channel/v1"
	"github.com/lisoboss/grpchub/grpchublog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

func defaultGrpchubDialOptionsWithTLS(caPEMBlock, certPEMBlock, keyPEMBlock []byte) ([]grpc.DialOption, error) {
	// 加载客户端证书和私钥
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// 加载 CA 证书
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEMBlock) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	// 构造 TLS 配置
	tlsConfig := &tls.Config{
		ServerName:         "localhost",
		Certificates:       []tls.Certificate{cert}, // 客户端证书和私钥
		RootCAs:            certPool,                // 服务端证书校验
		InsecureSkipVerify: false,                   // 开启主机名校验
	}

	// 使用 TLS 凭证连接
	creds := credentials.NewTLS(tlsConfig)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("zstd")),
	}
	return opts, nil
}
