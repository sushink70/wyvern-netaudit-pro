use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Serialize, Deserialize, FromRow)]
pub struct PingHistory {
    pub id: i32,
    pub ip_address: String,
    pub timestamp: DateTime<Utc>,
    pub is_successful: bool,
    pub min_latency: Option<f64>,
    pub avg_latency: Option<f64>,
    pub max_latency: Option<f64>,
    pub packet_loss: Option<f64>,
    pub packets_transmitted: Option<i32>,
    pub packets_received: Option<i32>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PingStats {
    pub min_latency: f64,
    pub avg_latency: f64,
    pub max_latency: f64,
    pub packet_loss: Option<f64>,
    pub packets_transmitted: Option<i32>,
    pub packets_received: Option<i32>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PingResult {
    pub success: bool,
    pub message: String,
    pub details: Option<PingStats>,
    pub raw_output: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, FromRow)]
pub struct TracerouteHistory {
    pub id: i32,
    pub ip_address: String,
    pub timestamp: DateTime<Utc>,
    pub is_successful: bool,
    pub total_hops: Option<i32>,
    pub completion_time: Option<f64>,
}

#[derive(Debug, Serialize, Deserialize, FromRow)]
pub struct TracerouteHop {
    pub id: i32,
    pub traceroute_id: i32,
    pub hop_number: i32,
    pub hostname: Option<String>,
    pub ip_address: Option<String>,
    pub rtt1: Option<f64>,
    pub rtt2: Option<f64>,
    pub rtt3: Option<f64>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct TracerouteResult {
    pub success: bool,
    pub message: String,
    pub parsed_hops: Option<Vec<TracerouteHop>>,
    pub total_hops: Option<i32>,
    pub completion_time: f64,
    pub raw_output: Option<String>,
}

impl PingHistory {
    pub async fn create(
        pool: &sqlx::PgPool,
        ip_address: &str,
        is_successful: bool,
        stats: Option<&PingStats>,
    ) -> Result<Self, sqlx::Error> {
        let (min_lat, avg_lat, max_lat, packet_loss, packets_tx, packets_rx) = 
            if let Some(stats) = stats {
                (
                    Some(stats.min_latency),
                    Some(stats.avg_latency),
                    Some(stats.max_latency),
                    stats.packet_loss,
                    stats.packets_transmitted,
                    stats.packets_received,
                )
            } else {
                (None, None, None, None, None, None)
            };

        sqlx::query_as!(
            PingHistory,
            r#"
            INSERT INTO ping_history 
            (ip_address, is_successful, min_latency, avg_latency, max_latency, 
             packet_loss, packets_transmitted, packets_received)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING *
            "#,
            ip_address,
            is_successful,
            min_lat,
            avg_lat,
            max_lat,
            packet_loss,
            packets_tx,
            packets_rx
        )
        .fetch_one(pool)
        .await
    }

    pub async fn get_recent(pool: &sqlx::PgPool, limit: i64) -> Result<Vec<Self>, sqlx::Error> {
        sqlx::query_as!(
            PingHistory,
            "SELECT * FROM ping_history ORDER BY timestamp DESC LIMIT $1",
            limit
        )
        .fetch_all(pool)
        .await
    }
}

impl TracerouteHistory {
    pub async fn create(
        pool: &sqlx::PgPool,
        ip_address: &str,
        is_successful: bool,
        total_hops: Option<i32>,
        completion_time: f64,
    ) -> Result<Self, sqlx::Error> {
        sqlx::query_as!(
            TracerouteHistory,
            r#"
            INSERT INTO traceroute_history 
            (ip_address, is_successful, total_hops, completion_time)
            VALUES ($1, $2, $3, $4)
            RETURNING *
            "#,
            ip_address,
            is_successful,
            total_hops,
            completion_time
        )
        .fetch_one(pool)
        .await
    }

    pub async fn get_recent(pool: &sqlx::PgPool, limit: i64) -> Result<Vec<Self>, sqlx::Error> {
        sqlx::query_as!(
            TracerouteHistory,
            "SELECT * FROM traceroute_history ORDER BY timestamp DESC LIMIT $1",
            limit
        )
        .fetch_all(pool)
        .await
    }
}

impl TracerouteHop {
    pub async fn create_batch(
        pool: &sqlx::PgPool,
        traceroute_id: i32,
        hops: &[TracerouteHop],
    ) -> Result<(), sqlx::Error> {
        for hop in hops {
            sqlx::query!(
                r#"
                INSERT INTO traceroute_hops 
                (traceroute_id, hop_number, hostname, ip_address, rtt1, rtt2, rtt3)
                VALUES ($1, $2, $3, $4, $5, $6, $7)
                "#,
                traceroute_id,
                hop.hop_number,
                hop.hostname,
                hop.ip_address,
                hop.rtt1,
                hop.rtt2,
                hop.rtt3
            )
            .execute(pool)
            .await?;
        }
        Ok(())
    }
}