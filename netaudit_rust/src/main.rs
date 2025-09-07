use actix_files::Files;
use actix_web::{middleware::Logger, web, App, HttpServer};
use sqlx::postgres::PgPoolOptions;
use std::env;
use tera::Tera;

mod models;
mod handlers;
mod utils;

pub struct AppState {
    pub db: sqlx::PgPool,
    pub templates: tera::Tera,
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    dotenv::dotenv().ok();
    env_logger::init();

    // Database connection
    let database_url = env::var("DATABASE_URL")
        .expect("DATABASE_URL must be set");
    
    let pool = PgPoolOptions::new()
        .max_connections(5)
        .connect(&database_url)
        .await
        .expect("Failed to create database pool");

    // Run migrations
    sqlx::migrate!("./migrations")
        .run(&pool)
        .await
        .expect("Failed to run migrations");

    // Initialize Tera templates
    let tera = Tera::new("templates/**/*.html")
        .expect("Failed to initialize Tera templates");

    let app_state = web::Data::new(AppState {
        db: pool,
        templates: tera,
    });

    log::info!("Starting NetAudit Pro server on http://127.0.0.1:8080");

    HttpServer::new(move || {
        App::new()
            .app_data(app_state.clone())
            .wrap(Logger::default())
            .route("/", web::get().to(handlers::dashboard))
            .route("/ping", web::get().to(handlers::ping_page))
            .route("/ping", web::post().to(handlers::ping_execute))
            .route("/traceroute", web::get().to(handlers::traceroute_page))
            .route("/traceroute", web::post().to(handlers::traceroute_execute))
            .service(Files::new("/static", "./static").show_files_listing())
    })
    .bind("127.0.0.1:8080")?
    .run()
    .await
}