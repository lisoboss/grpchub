mod server;

use std::net::ToSocketAddrs;
use tonic::transport::Server;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = "[::1]:50055".to_socket_addrs().unwrap().next().unwrap();
    println!("Listening: {addr}");

    Server::builder()
        .add_service(server::new_reflection_service())
        .add_service(server::new_service())
        .serve(addr)
        .await
        .unwrap();

    Ok(())
}
