mod server;

use std::{cell::LazyCell, env, net::ToSocketAddrs};
use tonic::transport::Server;

const ADDR: LazyCell<String> = LazyCell::new(|| match env::var("GRPCHUB_ADDR") {
    Ok(val) => val,
    _ => String::from("[::1]:50055"),
});

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = ADDR.to_socket_addrs().unwrap().next().unwrap();
    println!("Listening: {addr}");

    Server::builder()
        .add_service(server::new_reflection_service())
        .add_service(server::new_service())
        .serve(addr)
        .await
        .unwrap();

    Ok(())
}
