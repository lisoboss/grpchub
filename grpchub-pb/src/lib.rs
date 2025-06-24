pub mod google {
    pub mod protobuf {
        #![allow(clippy::doc_overindented_list_items)]
        tonic::include_proto!("google.protobuf");
    }

    pub mod rpc {
        #![allow(clippy::doc_overindented_list_items)]
        tonic::include_proto!("google.rpc");
    }
}

pub mod grpchub {
    pub mod channel {
        tonic::include_proto!("channel.v1");
    }

    pub const FILE_DESCRIPTOR_SET: &[u8] =
        tonic::include_file_descriptor_set!("grpchub_descriptor");
}
