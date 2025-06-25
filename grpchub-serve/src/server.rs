use dashmap::DashMap;
use grpchub_pb::grpchub::{
    self,
    channel::{self, ChannelMessage},
};
use std::{pin::Pin, sync::Arc};
use tokio::sync::mpsc;
use tokio_stream::{Stream, StreamExt, wrappers::ReceiverStream};
use tonic::{Request, Response, Status, Streaming, codec::CompressionEncoding};
use tonic_health::pb::health_server::{Health, HealthServer};

type ChannelResult<T> = Result<Response<T>, Status>;
type ChannelStream = Pin<Box<dyn Stream<Item = Result<channel::ChannelMessage, Status>> + Send>>;
type ChannelMap = Arc<DashMap<String, mpsc::Sender<Result<channel::ChannelMessage, Status>>>>;

#[derive(Debug)]
pub struct ChannelServer {
    channels: ChannelMap,
}

impl ChannelServer {
    pub fn new() -> Self {
        Self {
            channels: Arc::new(DashMap::new()),
        }
    }
}

#[tonic::async_trait]
impl channel::channel_service_server::ChannelService for ChannelServer {
    type ChannelStream = ChannelStream;

    async fn channel(
        &self,
        request: Request<Streaming<channel::ChannelMessage>>,
    ) -> ChannelResult<Self::ChannelStream> {
        let md = request.metadata();
        let sender_id = parse_metadata_value(md, "sender_id")?.to_string();
        let receiver_id = parse_metadata_value(md, "receiver_id")?.to_string();

        let mut stream = request.into_inner();
        // 初始化通道
        let (tx, rx) = mpsc::channel(32);

        // 注册信息
        self.channels.insert(sender_id.clone(), tx.clone());
        println!("Client connected: {}", sender_id);

        let channels = self.channels.clone();
        tokio::spawn(async move {
            while let Some(Ok(msg)) = stream.next().await {
                if let Some(tx) = channels.get(&receiver_id) {
                    let t = match &msg.pkg {
                        Some(pkg) => pkg.r#type(),
                        _ => channel::PackageType::PtUnknown,
                    };
                    println!("send {}({}) => {}", msg.sid, t.as_str_name(), receiver_id);
                    let _ = tx.send(Ok(msg)).await;
                } else {
                    let _ = tx
                        .send(Ok(ChannelMessage {
                            sid: msg.sid,
                            pkg: Some(new_error_not_found()),
                        }))
                        .await;
                    break;
                }
            }

            // 下线清理
            channels.remove(&sender_id);
            println!("Client disconnected: {}", sender_id);
        });

        Ok(Response::new(
            Box::pin(ReceiverStream::new(rx)) as Self::ChannelStream
        ))
    }
}

fn new_error_not_found() -> channel::MessagePackage {
    use prost::Message;

    let status = grpchub_pb::google::rpc::Status {
        code: 14, // UNAVAILABLE
        message: "target service is offline or not available".to_string(),
        details: vec![],
    };

    let mut buf = Vec::new();
    status.encode(&mut buf).unwrap();

    channel::MessagePackage {
        r#type: channel::PackageType::PtError as i32,
        method: String::new(),
        payload: Some(grpchub_pb::google::protobuf::Any {
            type_url: "type.googleapis.com/google.rpc.Status".to_string(),
            value: buf,
        }),
        md: Vec::new(),
    }
}

fn parse_metadata_value<'a>(
    md: &'a tonic::metadata::MetadataMap,
    key: &'a str,
) -> Result<&'a str, Status> {
    match md.get(key) {
        Some(v) => v
            .to_str()
            .map_err(|e| Status::invalid_argument(format!("{key} to str err in metadata: {e}"))),
        _ => Err(Status::invalid_argument(format!("No {key} in metadata"))),
    }
}

pub fn new_service() -> channel::channel_service_server::ChannelServiceServer<ChannelServer> {
    let server = ChannelServer::new();
    let server = channel::channel_service_server::ChannelServiceServer::new(server)
        .send_compressed(CompressionEncoding::Zstd)
        .accept_compressed(CompressionEncoding::Zstd);

    server
}

pub fn new_reflection_service() -> tonic_reflection::server::v1::ServerReflectionServer<
    impl tonic_reflection::server::v1::ServerReflection,
> {
    tonic_reflection::server::Builder::configure()
        .register_encoded_file_descriptor_set(grpchub::FILE_DESCRIPTOR_SET)
        .build_v1()
        .unwrap()
}

pub async fn new_health_service() -> HealthServer<impl Health> {
    let (health_reporter, health_service) = tonic_health::server::health_reporter();

    health_reporter
        .set_serving::<channel::channel_service_server::ChannelServiceServer<ChannelServer>>()
        .await;

    health_service
}
