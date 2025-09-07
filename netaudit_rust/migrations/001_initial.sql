-- Create ping_history table
CREATE TABLE IF NOT EXISTS ping_history (
    id SERIAL PRIMARY KEY,
    ip_address VARCHAR(45) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_successful BOOLEAN NOT NULL,
    min_latency FLOAT,
    avg_latency FLOAT,
    max_latency FLOAT,
    packet_loss FLOAT,
    packets_transmitted INTEGER,
    packets_received INTEGER
);

-- Create traceroute_history table
CREATE TABLE IF NOT EXISTS traceroute_history (
    id SERIAL PRIMARY KEY,
    ip_address VARCHAR(45) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_successful BOOLEAN NOT NULL,
    total_hops INTEGER,
    completion_time FLOAT
);

-- Create traceroute_hops table
CREATE TABLE IF NOT EXISTS traceroute_hops (
    id SERIAL PRIMARY KEY,
    traceroute_id INTEGER NOT NULL REFERENCES traceroute_history(id) ON DELETE CASCADE,
    hop_number INTEGER NOT NULL,
    hostname VARCHAR(255),
    ip_address VARCHAR(45),
    rtt1 FLOAT,
    rtt2 FLOAT,
    rtt3 FLOAT
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_ping_history_timestamp ON ping_history(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ping_history_ip ON ping_history(ip_address);

CREATE INDEX IF NOT EXISTS idx_traceroute_history_timestamp ON traceroute_history(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_traceroute_history_ip ON traceroute_history(ip_address);

CREATE INDEX IF NOT EXISTS idx_traceroute_hops_traceroute_id ON traceroute_hops(traceroute_id);
CREATE INDEX IF NOT EXISTS idx_traceroute_hops_hop_number ON traceroute_hops(hop_number);