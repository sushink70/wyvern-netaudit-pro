use actix_web::{web, HttpResponse, Result};
use serde::Deserialize;
use tera::Context;

use crate::models::*;
use crate::utils::{ping_device, traceroute_device};
use crate::AppState;

#[derive(Deserialize)]
pub struct PingRequest {
    target_ip: String,
}

#[derive(Deserialize)]
pub struct TracerouteRequest {
    target_ip: String,
}

pub async fn dashboard(data: web::Data<AppState>) -> Result<HttpResponse> {
    let context = Context::new();
    
    match data.templates.render("dashboard.html", &context) {
        Ok(html) => Ok(HttpResponse::Ok().content_type("text/html").body(html)),
        Err(e) => {
            log::error!("Template rendering error: {}", e);
            Ok(HttpResponse::InternalServerError().body("Template error"))
        }
    }
}

pub async fn ping_page(data: web::Data<AppState>) -> Result<HttpResponse> {
    let recent_pings = match PingHistory::get_recent(&data.db, 10).await {
        Ok(pings) => pings,
        Err(e) => {
            log::error!("Database error: {}", e);
            vec![]
        }
    };

    let mut context = Context::new();
    context.insert("recent_pings", &recent_pings);
    
    match data.templates.render("ping.html", &context) {
        Ok(html) => Ok(HttpResponse::Ok().content_type("text/html").body(html)),
        Err(e) => {
            log::error!("Template rendering error: {}", e);
            Ok(HttpResponse::InternalServerError().body("Template error"))
        }
    }
}

pub async fn ping_execute(
    data: web::Data<AppState>,
    form: web::Form<PingRequest>,
) -> Result<HttpResponse> {
    if form.target_ip.is_empty() {
        return Ok(HttpResponse::BadRequest().json(PingResult {
            success: false,
            message: "Please provide a valid IP address.".to_string(),
            details: None,
            raw_output: None,
        }));
    }

    let result = ping_device(&form.target_ip).await;
    
    // Save to database
    let stats = result.details.as_ref();
    if let Err(e) = PingHistory::create(&data.db, &form.target_ip, result.success, stats).await {
        log::error!("Failed to save ping history: {}", e);
    }

    Ok(HttpResponse::Ok().json(result))
}

pub async fn traceroute_page(data: web::Data<AppState>) -> Result<HttpResponse> {
    let recent_traceroutes = match TracerouteHistory::get_recent(&data.db, 10).await {
        Ok(traces) => traces,
        Err(e) => {
            log::error!("Database error: {}", e);
            vec![]
        }
    };

    let mut context = Context::new();
    context.insert("recent_traceroutes", &recent_traceroutes);
    
    match data.templates.render("traceroute.html", &context) {
        Ok(html) => Ok(HttpResponse::Ok().content_type("text/html").body(html)),
        Err(e) => {
            log::error!("Template rendering error: {}", e);
            Ok(HttpResponse::InternalServerError().body("Template error"))
        }
    }
}

pub async fn traceroute_execute(
    data: web::Data<AppState>,
    form: web::Form<TracerouteRequest>,
) -> Result<HttpResponse> {
    if form.target_ip.is_empty() {
        return Ok(HttpResponse::BadRequest().json(TracerouteResult {
            success: false,
            message: "Please provide a valid IP address.".to_string(),
            parsed_hops: None,
            total_hops: None,
            completion_time: 0.0,
            raw_output: None,
        }));
    }

    let result = traceroute_device(&form.target_ip).await;
    
    // Save to database
    let total_hops = result.total_hops;
    let history = TracerouteHistory::create(
        &data.db,
        &form.target_ip,
        result.success,
        total_hops,
        result.completion_time,
    ).await;

    if let (Ok(history), Some(hops)) = (history, &result.parsed_hops) {
        // Convert and save hops
        let db_hops: Vec<TracerouteHop> = hops
            .iter()
            .map(|h| TracerouteHop {
                id: 0, // Will be set by database
                traceroute_id: history.id,
                hop_number: h.hop_number,
                hostname: h.hostname.clone(),
                ip_address: h.ip_address.clone(),
                rtt1: h.rtt1,
                rtt2: h.rtt2,
                rtt3: h.rtt3,
            })
            .collect();

        if let Err(e) = TracerouteHop::create_batch(&data.db, history.id, &db_hops).await {
            log::error!("Failed to save traceroute hops: {}", e);
        }
    }

    Ok(HttpResponse::Ok().json(result))
}