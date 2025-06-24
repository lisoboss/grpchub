#!/bin/bash

# TLS证书生成和合并脚本
# 用于生成服务端和客户端的PEM文件

set -e  # 遇到错误时退出

echo "🔐 开始生成TLS证书和PEM文件..."

# 创建证书目录
mkdir -p certs
cd certs

# 清理旧文件
rm -f *.pem *.crt *.key *.csr *.srl

echo "📋 复制配置文件..."
cp -p ../cert-conf/*.conf .

echo "🔑 1. 生成CA根证书..."
# 生成CA私钥
openssl genrsa -out ca.key 4096

# 生成CA根证书
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt -config ca.conf

echo "🖥️  2. 生成服务器证书..."
# 生成服务器私钥
openssl genrsa -out server.key 2048

# 生成服务器证书签名请求
openssl req -new -key server.key -out server.csr -config server.conf

# 用CA签名生成服务器证书
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -extensions v3_req -extfile server.conf

echo "👤 3. 生成客户端证书..."
# 生成客户端私钥
openssl genrsa -out client.key 2048

# 生成客户端证书签名请求
openssl req -new -key client.key -out client.csr -config client.conf

# 用CA签名生成客户端证书
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365 -extensions v3_req -extfile client.conf

echo "📦 4. 生成合并的PEM文件..."

# 生成服务端PEM文件（包含服务器证书、私钥和CA证书）
echo "# Server certificate and private key" > server.pem
cat server.crt >> server.pem
echo "" >> server.pem
cat server.key >> server.pem
echo "" >> server.pem
echo "# CA root certificate (used to verify the client)" >> server.pem
cat ca.crt >> server.pem

# 生成客户端PEM文件（包含客户端证书、私钥和CA证书）
echo "# Client certificate and private key" > client.pem
cat client.crt >> client.pem
echo "" >> client.pem
cat client.key >> client.pem
echo "" >> client.pem
echo "# CA root certificate (used to verify the server)" >> client.pem
cat ca.crt >> client.pem

echo "🧹 5. 清理临时文件..."
rm -f *.csr *.srl *.conf

echo "✅ 证书生成完成！"
echo ""
echo "📁 生成的文件："
echo "   📄 ca.crt      - CA根证书"
echo "   🔐 ca.key      - CA私钥"
echo "   📄 server.crt  - 服务器证书" 
echo "   🔐 server.key  - 服务器私钥"
echo "   📄 client.crt  - 客户端证书"
echo "   🔐 client.key  - 客户端私钥"
echo "   📦 server.pem  - 服务端合并PEM文件"
echo "   📦 client.pem  - 客户端合并PEM文件"
echo ""
echo "🚀 使用方法："
echo "   服务端：使用 server.pem"
echo "   客户端：使用 client.pem"
echo ""
echo "🔍 验证证书："
echo "   openssl x509 -in server.crt -text -noout"
echo "   openssl x509 -in client.crt -text -noout"
echo ""
echo "🧪 测试连接："
echo "   # 启动测试服务器"
echo "   openssl s_server -accept 8443 -cert server.crt -key server.key -CAfile ca.crt -verify 1"
echo "   # 客户端连接测试"  
echo "   openssl s_client -connect localhost:8443 -cert client.crt -key client.key -CAfile ca.crt"