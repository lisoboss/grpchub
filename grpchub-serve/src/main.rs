mod server;

use clap::Parser;
use std::{fs, net::ToSocketAddrs, path::PathBuf};
use tonic::transport::{Certificate, Identity, Server, ServerTlsConfig};

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
struct Args {
    /// Addr of the program to listen addr
    #[arg(short, long, default_value = "[::1]:50055")]
    addr: String,

    /// Pem of the TLS PEM file path
    #[arg(short, long, default_value = "./server.pem")]
    pem: PathBuf,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let Args { addr, pem } = Args::parse();
    let addr = addr.to_socket_addrs().unwrap().next().unwrap();
    println!("Listening: {addr}");

    let pem = fs::read(pem)?;
    let identity = Identity::from_pem(&pem, &pem);
    let ca_cert = Certificate::from_pem(&pem);

    let tls = ServerTlsConfig::new()
        .identity(identity)
        .client_ca_root(ca_cert);

    Server::builder()
        .tls_config(tls)?
        .add_service(server::new_service())
        .add_service(server::new_health_service().await)
        .add_service(server::new_reflection_service())
        .serve(addr)
        .await
        .unwrap();

    Ok(())
}
