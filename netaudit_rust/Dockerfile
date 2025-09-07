# Build stage
FROM rust:1.75 as builder

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src/ src/
COPY migrations/ migrations/
COPY templates/ templates/

RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

# Install required system packages
RUN apt-get update && apt-get install -y \
    iputils-ping \
    traceroute \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary and templates
COPY --from=builder /app/target/release/netaudit-pro /usr/local/bin/netaudit-pro
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/migrations ./migrations

# Create static directory
RUN mkdir -p static

EXPOSE 8080

CMD ["netaudit-pro"]