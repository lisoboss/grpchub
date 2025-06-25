FROM rust:1.87.0-bookworm AS builder
RUN apt-get update && apt-get install -y protobuf-compiler 
WORKDIR /usr/src/build
COPY . .
RUN cargo build --release

FROM debian:bookworm-slim AS prod
RUN rm -rf /var/lib/apt/lists/*
WORKDIR /usr/app
COPY --from=builder /usr/src/build/target/release/grpchub-serve /usr/local/bin/grpchub-serve
EXPOSE 50055
CMD ["grpchub-serve"]