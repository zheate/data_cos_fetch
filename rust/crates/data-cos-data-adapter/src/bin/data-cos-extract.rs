//! A one-request extraction worker. stdout is reserved for the JSON result.
use std::io::{self, Read, Write};

use anyhow::{Context, Result, bail};
use data_cos_data_adapter::{DataFetchExtractRequest, extract_data};
use serde_json::json;

fn run() -> Result<()> {
    let mut input = String::new();
    io::stdin()
        .take(16 * 1024 * 1024 + 1)
        .read_to_string(&mut input)?;
    if input.len() > 16 * 1024 * 1024 {
        bail!("request exceeds 16 MiB");
    }
    let request: DataFetchExtractRequest =
        serde_json::from_str(&input).context("invalid extraction request")?;
    if request.entries.is_empty() {
        bail!("at least one entry is required");
    }
    let result = extract_data(&request)?;
    serde_json::to_writer(
        io::stdout().lock(),
        &json!({
            "total": result.records.len(),
            "records": result.records,
            "errors": result.errors,
            "infos": result.infos,
        }),
    )?;
    io::stdout().flush()?;
    Ok(())
}

fn main() {
    if let Err(error) = run() {
        eprintln!("{error:#}");
        std::process::exit(1);
    }
}
