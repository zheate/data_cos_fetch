use std::collections::{BTreeMap, HashMap, HashSet};

use serde::{Deserialize, Serialize};
use thiserror::Error;

pub const WAREHOUSE_AVAILABLE: [&str; 1] = ["良品仓"];
pub const WAREHOUSE_NEED_CONFIRM: [&str; 3] = ["研发工程仓", "生产验证仓", "报废1仓"];

#[derive(Debug, Error)]
pub enum CoreError {
    #[error("group_size must be greater than 0")]
    InvalidGroupSize,
    #[error("flat-top requires max_diff_nm > 0")]
    InvalidFlatTopConfig,
    #[error("huang-meng requires avg_min_nm and avg_max_nm")]
    MissingAverageRange,
    #[error("huang-meng requires low/high range fields")]
    MissingHuangRange,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum WavelengthField {
    Cold,
    Center,
    TwoA,
    Peak,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CosRecord {
    pub device_id: String,
    pub warehouse: Option<String>,
    pub isolation: Option<String>,
    pub item_num: Option<String>,
    pub box_num: Option<String>,
    pub owner: Option<String>,
    pub cold_wavelength_nm: Option<f64>,
    pub center_wavelength_nm: Option<f64>,
    pub two_a_wavelength_nm: Option<f64>,
    pub peak_wavelength_nm: Option<f64>,
}

impl CosRecord {
    pub fn wavelength_value(&self, field: WavelengthField) -> Option<f64> {
        let value = match field {
            WavelengthField::Cold => self.cold_wavelength_nm,
            WavelengthField::Center => self.center_wavelength_nm,
            WavelengthField::TwoA => self.two_a_wavelength_nm,
            WavelengthField::Peak => self.peak_wavelength_nm,
        };
        value.filter(|v| v.is_finite())
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CosStep1Request {
    pub wavelength_field: WavelengthField,
    pub wavelength_min_nm: f64,
    pub wavelength_max_nm: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CosStep2Request {
    pub wavelength_field: WavelengthField,
    pub wavelength_min_nm: f64,
    pub wavelength_max_nm: f64,
    pub required_count: usize,
    pub item_num_filter: Option<Vec<String>>,
    pub box_num_filter: Option<Vec<String>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CosGroupingRequest {
    pub wavelength_field: WavelengthField,
    pub group_size: usize,
    pub max_diff_nm: f64,
    pub avg_min_nm: Option<f64>,
    pub avg_max_nm: Option<f64>,
    pub strict_mode: Option<bool>,
    /// 均匀性 CV 阈值（仅 strict_mode=true 时生效），默认 0.8
    pub uniformity_cv: Option<f64>,
    pub low_min_nm: Option<f64>,
    pub low_max_nm: Option<f64>,
    pub high_min_nm: Option<f64>,
    pub high_max_nm: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CosGroupingResult {
    pub groups: Vec<Vec<CosRecord>>,
    pub remaining: Vec<CosRecord>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct DataFetchRecord {
    pub entry_id: String,
    pub test_category: String,
    pub current_a: Option<f64>,
    pub power_w: Option<f64>,
    pub voltage_v: Option<f64>,
    pub efficiency_pct: Option<f64>,
    pub lambda_nm: Option<f64>,
    pub shift_nm: Option<f64>,
    pub wavelength_2a_nm: Option<f64>,
    pub wavelength_cold_nm: Option<f64>,
}

#[derive(Debug, Clone)]
struct ScoredRecord {
    record: CosRecord,
    distance: f64,
}

pub fn filter_cos_step1(records: &[CosRecord], req: &CosStep1Request) -> Vec<CosRecord> {
    let mut min_nm = req.wavelength_min_nm;
    let mut max_nm = req.wavelength_max_nm;
    if min_nm > max_nm {
        std::mem::swap(&mut min_nm, &mut max_nm);
    }

    records
        .iter()
        .filter(|record| matches!(record.isolation.as_deref(), Some("否")))
        .filter(|record| {
            record
                .warehouse
                .as_deref()
                .map(|warehouse| {
                    WAREHOUSE_AVAILABLE.contains(&warehouse)
                        || WAREHOUSE_NEED_CONFIRM.contains(&warehouse)
                })
                .unwrap_or(false)
        })
        .filter(|record| {
            record
                .wavelength_value(req.wavelength_field)
                .map(|wavelength| wavelength >= min_nm && wavelength <= max_nm)
                .unwrap_or(false)
        })
        .cloned()
        .collect()
}

pub fn filter_cos_step2(records: &[CosRecord], req: &CosStep2Request) -> Vec<CosRecord> {
    if req.required_count == 0 {
        return Vec::new();
    }

    let mut min_nm = req.wavelength_min_nm;
    let mut max_nm = req.wavelength_max_nm;
    if min_nm > max_nm {
        std::mem::swap(&mut min_nm, &mut max_nm);
    }
    let target = (min_nm + max_nm) / 2.0;

    let filtered: Vec<CosRecord> = records
        .iter()
        .filter(|record| {
            req.item_num_filter
                .as_ref()
                .map(|selected| {
                    if selected.is_empty() {
                        return true;
                    }
                    record
                        .item_num
                        .as_ref()
                        .map(|item_num| {
                            selected
                                .iter()
                                .any(|selected_item| selected_item == item_num)
                        })
                        .unwrap_or(false)
                })
                .unwrap_or(true)
        })
        .filter(|record| {
            req.box_num_filter
                .as_ref()
                .map(|selected| {
                    if selected.is_empty() {
                        return true;
                    }
                    record
                        .box_num
                        .as_ref()
                        .map(|box_num| selected.iter().any(|selected_box| selected_box == box_num))
                        .unwrap_or(false)
                })
                .unwrap_or(true)
        })
        .cloned()
        .collect();

    let mut scored: Vec<ScoredRecord> = filtered
        .into_iter()
        .filter_map(|record| {
            record
                .wavelength_value(req.wavelength_field)
                .map(|wavelength| ScoredRecord {
                    distance: (wavelength - target).abs(),
                    record,
                })
        })
        .collect();

    if scored.is_empty() {
        let fallback: Vec<CosRecord> = records
            .iter()
            .filter(|record| {
                req.item_num_filter
                    .as_ref()
                    .map(|selected| {
                        if selected.is_empty() {
                            return true;
                        }
                        record
                            .item_num
                            .as_ref()
                            .map(|item_num| {
                                selected
                                    .iter()
                                    .any(|selected_item| selected_item == item_num)
                            })
                            .unwrap_or(false)
                    })
                    .unwrap_or(true)
            })
            .filter(|record| {
                req.box_num_filter
                    .as_ref()
                    .map(|selected| {
                        if selected.is_empty() {
                            return true;
                        }
                        record
                            .box_num
                            .as_ref()
                            .map(|box_num| {
                                selected.iter().any(|selected_box| selected_box == box_num)
                            })
                            .unwrap_or(false)
                    })
                    .unwrap_or(true)
            })
            .take(req.required_count)
            .cloned()
            .collect();
        return fallback;
    }

    let mut available: Vec<ScoredRecord> = Vec::new();
    let mut need_confirm: Vec<ScoredRecord> = Vec::new();

    for entry in scored.drain(..) {
        match entry.record.warehouse.as_deref() {
            Some(warehouse) if WAREHOUSE_AVAILABLE.contains(&warehouse) => available.push(entry),
            Some(warehouse) if WAREHOUSE_NEED_CONFIRM.contains(&warehouse) => {
                need_confirm.push(entry)
            }
            _ => {}
        }
    }

    let mut selected = Vec::new();
    selected.extend(select_with_min_box(&available, req.required_count));

    if selected.len() < req.required_count {
        let remaining = req.required_count - selected.len();
        selected.extend(select_with_min_box(&need_confirm, remaining));
    }

    selected
}

fn select_with_min_box(pool: &[ScoredRecord], required_count: usize) -> Vec<CosRecord> {
    if required_count == 0 || pool.is_empty() {
        return Vec::new();
    }
    if pool.len() <= required_count {
        return pool.iter().map(|entry| entry.record.clone()).collect();
    }

    const OWNER_PRIORITY: [&str; 2] = ["模块事业部", "长光华芯"];

    let mut remaining = required_count;
    let mut selected = Vec::new();

    let by_owner: Vec<Vec<&ScoredRecord>> = OWNER_PRIORITY
        .iter()
        .map(|owner| {
            pool.iter()
                .filter(|entry| entry.record.owner.as_deref() == Some(*owner))
                .collect()
        })
        .collect();

    for owner_pool in by_owner {
        if remaining == 0 {
            break;
        }
        let picked = pick_by_box(owner_pool, remaining);
        remaining = remaining.saturating_sub(picked.len());
        selected.extend(picked);
    }

    if remaining > 0 {
        let other_pool: Vec<&ScoredRecord> = pool
            .iter()
            .filter(|entry| {
                !OWNER_PRIORITY
                    .iter()
                    .any(|owner| entry.record.owner.as_deref() == Some(*owner))
            })
            .collect();
        let picked = pick_by_box(other_pool, remaining);
        selected.extend(picked);
    }

    selected
}

fn pick_by_box(pool: Vec<&ScoredRecord>, required_count: usize) -> Vec<CosRecord> {
    if required_count == 0 || pool.is_empty() {
        return Vec::new();
    }

    let candidate_size = pool.len().min(required_count.saturating_mul(2));
    let mut ranked = pool;
    ranked.sort_by(|left, right| left.distance.total_cmp(&right.distance));
    ranked.truncate(candidate_size);

    let mut box_map: HashMap<String, Vec<&ScoredRecord>> = HashMap::new();
    for entry in ranked {
        let key = entry
            .record
            .box_num
            .clone()
            .unwrap_or_else(|| String::from("__missing_box__"));
        box_map.entry(key).or_default().push(entry);
    }

    let mut box_vec: Vec<(String, Vec<&ScoredRecord>)> = box_map.into_iter().collect();
    box_vec.sort_by(|left, right| {
        right
            .1
            .len()
            .cmp(&left.1.len())
            .then_with(|| left.0.cmp(&right.0))
    });

    let mut selected = Vec::new();
    let mut remaining = required_count;

    for (_, mut entries) in box_vec {
        if remaining == 0 {
            break;
        }
        entries.sort_by(|left, right| left.distance.total_cmp(&right.distance));
        let take = entries.len().min(remaining);
        for entry in entries.into_iter().take(take) {
            selected.push(entry.record.clone());
        }
        remaining = remaining.saturating_sub(take);
    }

    selected
}

pub fn group_wavelengths_greedy(
    records: &[CosRecord],
    req: &CosGroupingRequest,
) -> Result<CosGroupingResult, CoreError> {
    if req.group_size == 0 {
        return Err(CoreError::InvalidGroupSize);
    }

    let with_wavelength = collect_sorted_wavelength_records(records, req.wavelength_field);

    let mut groups_idx: Vec<Vec<usize>> = Vec::new();
    let mut i = 0usize;

    while i + req.group_size <= with_wavelength.len() {
        let window = &with_wavelength[i..i + req.group_size];
        let min_wl = window.first().map(|entry| entry.1).unwrap_or(0.0);
        let max_wl = window.last().map(|entry| entry.1).unwrap_or(0.0);

        let mean = window.iter().map(|entry| entry.1).sum::<f64>() / req.group_size as f64;
        let valid = (max_wl - min_wl) <= req.max_diff_nm
            && avg_constraints_match(mean, req.avg_min_nm, req.avg_max_nm);

        if valid {
            groups_idx.push((i..i + req.group_size).collect());
            i += req.group_size;
        } else {
            i += 1;
        }
    }

    Ok(build_grouping_result(&with_wavelength, groups_idx))
}

pub fn group_wavelengths_optimal(
    records: &[CosRecord],
    req: &CosGroupingRequest,
) -> Result<CosGroupingResult, CoreError> {
    if req.group_size == 0 {
        return Err(CoreError::InvalidGroupSize);
    }

    let with_wavelength = collect_sorted_wavelength_records(records, req.wavelength_field);
    if with_wavelength.len() < req.group_size {
        return Ok(CosGroupingResult {
            groups: Vec::new(),
            remaining: with_wavelength
                .into_iter()
                .map(|(record, _)| record)
                .collect(),
        });
    }

    let values: Vec<f64> = with_wavelength.iter().map(|entry| entry.1).collect();
    let target_center = if let (Some(avg_min), Some(avg_max)) = (req.avg_min_nm, req.avg_max_nm) {
        (avg_min + avg_max) / 2.0
    } else if let Some(avg_min) = req.avg_min_nm {
        avg_min
    } else if let Some(avg_max) = req.avg_max_nm {
        avg_max
    } else {
        values.iter().sum::<f64>() / values.len() as f64
    };

    let mut windows: Vec<(usize, usize, f64)> = Vec::new();
    for start in 0..=(values.len() - req.group_size) {
        let end = start + req.group_size - 1;
        let min_wl = values[start];
        let max_wl = values[end];
        if (max_wl - min_wl) > req.max_diff_nm {
            continue;
        }
        let mean = values[start..=end].iter().sum::<f64>() / req.group_size as f64;
        if !avg_constraints_match(mean, req.avg_min_nm, req.avg_max_nm) {
            continue;
        }
        let weight = 1.0 / (1.0 + (mean - target_center).abs());
        windows.push((start, end, weight));
    }

    if windows.is_empty() {
        return Ok(CosGroupingResult {
            groups: Vec::new(),
            remaining: with_wavelength
                .into_iter()
                .map(|(record, _)| record)
                .collect(),
        });
    }

    windows.sort_by(|left, right| left.1.cmp(&right.1));

    let mut prev_non_overlap: Vec<Option<usize>> = vec![None; windows.len()];
    for j in 0..windows.len() {
        let start_j = windows[j].0;
        // Use binary search (partition_point) to find the nearest non-overlapping window in O(log N)
        // Since `windows` is sorted by windows[i].1 (end index), partition_point perfectly finds 
        // the first window where end index >= start_j.
        let i = windows[..j].partition_point(|w| w.1 < start_j);
        if i > 0 {
            prev_non_overlap[j] = Some(i - 1);
        }
    }

    let mut dp = vec![0.0; windows.len() + 1];
    let mut choose = vec![false; windows.len()];
    for j in 0..windows.len() {
        let skip = dp[j];
        let take_base = prev_non_overlap[j].map(|idx| dp[idx + 1]).unwrap_or(0.0);
        let take = windows[j].2 + take_base;
        if take > skip {
            dp[j + 1] = take;
            choose[j] = true;
        } else {
            dp[j + 1] = skip;
        }
    }

    let mut selected_windows: Vec<usize> = Vec::new();
    let mut j = windows.len() as isize - 1;
    while j >= 0 {
        let current = j as usize;
        if choose[current] {
            selected_windows.push(current);
            if let Some(prev) = prev_non_overlap[current] {
                j = prev as isize;
            } else {
                break;
            }
        } else {
            j -= 1;
        }
    }
    selected_windows.reverse();

    let groups_idx: Vec<Vec<usize>> = selected_windows
        .into_iter()
        .map(|window_idx| {
            let (start, end, _) = windows[window_idx];
            (start..=end).collect()
        })
        .collect();

    Ok(build_grouping_result(&with_wavelength, groups_idx))
}

pub fn group_wavelengths_flat_top(
    records: &[CosRecord],
    req: &CosGroupingRequest,
) -> Result<CosGroupingResult, CoreError> {
    if req.group_size == 0 {
        return Err(CoreError::InvalidGroupSize);
    }
    if req.max_diff_nm <= 0.0 {
        return Err(CoreError::InvalidFlatTopConfig);
    }

    let with_wavelength = collect_sorted_wavelength_records(records, req.wavelength_field);
    let n_total = with_wavelength.len();
    let group_size = req.group_size;
    if n_total < group_size {
        return Ok(CosGroupingResult {
            groups: Vec::new(),
            remaining: with_wavelength
                .into_iter()
                .map(|(record, _)| record)
                .collect(),
        });
    }

    let values: Vec<f64> = with_wavelength.iter().map(|entry| entry.1).collect();
    let data_min = values[0];
    let data_max = values[n_total - 1];
    if (data_max - data_min) < 0.01 {
        return Ok(CosGroupingResult {
            groups: Vec::new(),
            remaining: with_wavelength
                .into_iter()
                .map(|(record, _)| record)
                .collect(),
        });
    }

    let (bin_min, bin_max) = if (data_max - data_min) > req.max_diff_nm {
        let mut best_start = 0usize;
        let mut max_count = 0usize;
        let mut right = 0usize;
        for left in 0..n_total {
            while right < n_total && values[right] <= values[left] + req.max_diff_nm {
                right += 1;
            }
            let count = right - left;
            if count > max_count {
                max_count = count;
                best_start = left;
            }
        }
        (values[best_start], values[best_start] + req.max_diff_nm)
    } else {
        (data_min, data_max)
    };

    let bin_width = (bin_max - bin_min) / group_size as f64;
    if bin_width <= f64::EPSILON {
        return Ok(CosGroupingResult {
            groups: Vec::new(),
            remaining: with_wavelength
                .into_iter()
                .map(|(record, _)| record)
                .collect(),
        });
    }

    let mut bins: Vec<Vec<(usize, f64)>> = vec![Vec::new(); group_size];
    for (index, (_, wl)) in with_wavelength.iter().enumerate() {
        if *wl < bin_min || *wl > bin_max {
            continue;
        }
        let mut bin_idx = ((*wl - bin_min) / bin_width) as usize;
        if bin_idx >= group_size {
            bin_idx = group_size - 1;
        }
        bins[bin_idx].push((index, *wl));
    }

    for (idx, entries) in bins.iter_mut().enumerate() {
        let center = bin_min + (idx as f64 + 0.5) * bin_width;
        entries.sort_by(|left, right| (left.1 - center).abs().total_cmp(&(right.1 - center).abs()));
    }

    let strict_mode = req.strict_mode.unwrap_or(false);
    let max_gap_cv = if strict_mode {
        req.uniformity_cv.unwrap_or(0.8)
    } else {
        999.0
    };
    let mut bin_cursors = vec![0usize; group_size];
    let mut used: HashSet<usize> = HashSet::new();
    let mut groups_idx: Vec<Vec<usize>> = Vec::new();

    for _ in 0..n_total {
        let mut selected_idx: Vec<usize> = Vec::with_capacity(group_size);
        let mut selected_wl: Vec<f64> = Vec::with_capacity(group_size);

        for bin_idx in 0..group_size {
            let chip = find_chip_with_neighbor(bin_idx, &bins, &mut bin_cursors, &used, group_size);
            if let Some((idx, wl)) = chip {
                selected_idx.push(idx);
                selected_wl.push(wl);
                used.insert(idx);
            } else {
                break;
            }
        }

        if selected_idx.len() < group_size {
            for idx in selected_idx {
                used.remove(&idx);
            }
            break;
        }

        if strict_mode && group_size > 2 {
            let mut sorted = selected_wl.clone();
            sorted.sort_by(|left, right| left.total_cmp(right));
            let gaps: Vec<f64> = sorted.windows(2).map(|pair| pair[1] - pair[0]).collect();
            let gap_mean = gaps.iter().sum::<f64>() / gaps.len() as f64;
            let gap_cv = if gap_mean > 0.0 {
                let var = gaps
                    .iter()
                    .map(|gap| {
                        let diff = *gap - gap_mean;
                        diff * diff
                    })
                    .sum::<f64>()
                    / gaps.len() as f64;
                var.sqrt() / gap_mean
            } else {
                0.0
            };

            if gap_cv > max_gap_cv {
                let (max_gap_idx, _) = gaps
                    .iter()
                    .enumerate()
                    .max_by(|left, right| left.1.total_cmp(right.1))
                    .unwrap_or((0, &0.0));
                let reject_idx = selected_idx[max_gap_idx + 1];
                for idx in selected_idx {
                    if idx != reject_idx {
                        used.remove(&idx);
                    }
                }
                continue;
            }
        }

        let mean = selected_wl.iter().sum::<f64>() / group_size as f64;
        if let Some(avg_min) = req.avg_min_nm {
            if mean < avg_min {
                let reject_idx = selected_idx[0];
                for idx in selected_idx {
                    if idx != reject_idx {
                        used.remove(&idx);
                    }
                }
                continue;
            }
        }
        if let Some(avg_max) = req.avg_max_nm {
            if mean > avg_max {
                let reject_idx = selected_idx[group_size - 1];
                for idx in selected_idx {
                    if idx != reject_idx {
                        used.remove(&idx);
                    }
                }
                continue;
            }
        }

        groups_idx.push(selected_idx);
    }

    Ok(build_grouping_result(&with_wavelength, groups_idx))
}

pub fn group_wavelengths_huang_meng(
    records: &[CosRecord],
    req: &CosGroupingRequest,
) -> Result<CosGroupingResult, CoreError> {
    if req.group_size == 0 {
        return Err(CoreError::InvalidGroupSize);
    }

    let (Some(avg_min), Some(avg_max)) = (req.avg_min_nm, req.avg_max_nm) else {
        return Err(CoreError::MissingAverageRange);
    };
    let (Some(low_min), Some(low_max), Some(high_min), Some(high_max)) = (
        req.low_min_nm,
        req.low_max_nm,
        req.high_min_nm,
        req.high_max_nm,
    ) else {
        return Err(CoreError::MissingHuangRange);
    };

    let with_wavelength = collect_sorted_wavelength_records(records, req.wavelength_field);
    if with_wavelength.is_empty() {
        return Ok(CosGroupingResult {
            groups: Vec::new(),
            remaining: Vec::new(),
        });
    }

    let mut low_band: Vec<(usize, f64)> = Vec::new();
    let mut high_band: Vec<(usize, f64)> = Vec::new();
    for (idx, (_, wl)) in with_wavelength.iter().enumerate() {
        if *wl >= low_min && *wl <= low_max {
            low_band.push((idx, *wl));
        }
        if *wl >= high_min && *wl <= high_max {
            high_band.push((idx, *wl));
        }
    }

    low_band.sort_by(|left, right| left.1.total_cmp(&right.1));
    high_band.sort_by(|left, right| left.1.total_cmp(&right.1));

    let half = req.group_size / 2;
    let (low_count, high_count) = if req.group_size % 2 == 0 {
        (half, half)
    } else {
        (half + 1, half)
    };

    if low_band.len() < low_count || high_band.len() < high_count {
        return Ok(CosGroupingResult {
            groups: Vec::new(),
            remaining: with_wavelength
                .into_iter()
                .map(|(record, _)| record)
                .collect(),
        });
    }

    let target_center = (avg_min + avg_max) / 2.0;
    let mut used_low: HashSet<usize> = HashSet::new();
    let mut used_high: HashSet<usize> = HashSet::new();
    let mut groups_idx: Vec<Vec<usize>> = Vec::new();

    let mut available_low: Vec<usize> = (0..low_band.len()).collect();
    let mut available_high: Vec<usize> = (0..high_band.len()).collect();

    loop {
        available_low.retain(|&idx| !used_low.contains(&low_band[idx].0));
        available_high.retain(|&idx| !used_high.contains(&high_band[idx].0));

        if available_low.len() < low_count || available_high.len() < high_count {
            break;
        }

        let selected_low_local = &available_low[0..low_count];
        let selected_low_wl: Vec<f64> = selected_low_local
            .iter()
            .map(|idx| low_band[*idx].1)
            .collect();
        let low_sum = selected_low_wl.iter().sum::<f64>();

        let mut best_high_window: Option<Vec<usize>> = None;
        let mut best_avg_diff = f64::INFINITY;

        for start in 0..=(available_high.len() - high_count) {
            let window = &available_high[start..start + high_count];
            let window_wl: Vec<f64> = window.iter().map(|idx| high_band[*idx].1).collect();
            let combined_sum = low_sum + window_wl.iter().sum::<f64>();
            let combined_avg = combined_sum / req.group_size as f64;

            if combined_avg < avg_min || combined_avg > avg_max {
                continue;
            }

            if req.max_diff_nm > 0.0 {
                let mut all_wl = selected_low_wl.clone();
                all_wl.extend(window_wl.iter().copied());
                let min_wl = all_wl
                    .iter()
                    .min_by(|left, right| left.total_cmp(right))
                    .copied()
                    .unwrap_or(0.0);
                let max_wl = all_wl
                    .iter()
                    .max_by(|left, right| left.total_cmp(right))
                    .copied()
                    .unwrap_or(0.0);
                if (max_wl - min_wl) > req.max_diff_nm {
                    continue;
                }
            }

            let avg_diff = (combined_avg - target_center).abs();
            if avg_diff < best_avg_diff {
                best_avg_diff = avg_diff;
                best_high_window = Some(window.to_vec());
            }
        }

        if let Some(best_high_local) = best_high_window {
            let mut group = Vec::with_capacity(req.group_size);
            for local in selected_low_local {
                let global_idx = low_band[*local].0;
                used_low.insert(global_idx);
                group.push(global_idx);
            }
            for local in best_high_local {
                let global_idx = high_band[local].0;
                used_high.insert(global_idx);
                group.push(global_idx);
            }
            groups_idx.push(group);
        } else if let Some(first_low) = available_low.first() {
            used_low.insert(low_band[*first_low].0);
        } else {
            break;
        }
    }

    Ok(build_grouping_result(&with_wavelength, groups_idx))
}

fn collect_sorted_wavelength_records(
    records: &[CosRecord],
    field: WavelengthField,
) -> Vec<(CosRecord, f64)> {
    let mut with_wavelength: Vec<(CosRecord, f64)> = records
        .iter()
        .filter_map(|record| {
            record
                .wavelength_value(field)
                .map(|wl| (record.clone(), wl))
        })
        .collect();
    with_wavelength.sort_by(|left, right| left.1.total_cmp(&right.1));
    with_wavelength
}

fn avg_constraints_match(value: f64, avg_min: Option<f64>, avg_max: Option<f64>) -> bool {
    if let Some(avg_min) = avg_min {
        if value < avg_min {
            return false;
        }
    }
    if let Some(avg_max) = avg_max {
        if value > avg_max {
            return false;
        }
    }
    true
}

fn build_grouping_result(
    with_wavelength: &[(CosRecord, f64)],
    groups_idx: Vec<Vec<usize>>,
) -> CosGroupingResult {
    let mut used_positions: HashSet<usize> = HashSet::new();
    let groups: Vec<Vec<CosRecord>> = groups_idx
        .into_iter()
        .map(|group| {
            let mut current = Vec::with_capacity(group.len());
            for position in group {
                used_positions.insert(position);
                current.push(with_wavelength[position].0.clone());
            }
            current
        })
        .collect();

    let remaining = with_wavelength
        .iter()
        .enumerate()
        .filter_map(|(position, (record, _))| {
            if used_positions.contains(&position) {
                None
            } else {
                Some(record.clone())
            }
        })
        .collect();

    CosGroupingResult { groups, remaining }
}

fn find_chip_with_neighbor(
    bin_idx: usize,
    bins: &[Vec<(usize, f64)>],
    cursors: &mut [usize],
    used: &HashSet<usize>,
    total_bins: usize,
) -> Option<(usize, f64)> {
    let mut cursor = cursors[bin_idx];
    while cursor < bins[bin_idx].len() && used.contains(&bins[bin_idx][cursor].0) {
        cursor += 1;
    }
    cursors[bin_idx] = cursor;
    if cursor < bins[bin_idx].len() {
        return Some(bins[bin_idx][cursor]);
    }

    for offset in 1..total_bins {
        for next in [bin_idx + offset, bin_idx.saturating_sub(offset)] {
            if next >= total_bins {
                continue;
            }
            let mut next_cursor = cursors[next];
            while next_cursor < bins[next].len() && used.contains(&bins[next][next_cursor].0) {
                next_cursor += 1;
            }
            cursors[next] = next_cursor;
            if next_cursor < bins[next].len() {
                return Some(bins[next][next_cursor]);
            }
        }
    }
    None
}

pub fn merge_data_fetch_records(records: &[DataFetchRecord]) -> Vec<DataFetchRecord> {
    let mut merged: BTreeMap<(String, String, i64), DataFetchRecord> = BTreeMap::new();

    for record in records {
        let Some(current_a) = record.current_a else {
            continue;
        };
        // 归一化到微安 (1e-6) 以进行稳定匹配
        let current_u_key = (current_a * 1_000_000.0).round() as i64;
        let key = (
            record.entry_id.clone(),
            record.test_category.clone(),
            current_u_key,
        );

        let entry = merged.entry(key).or_insert_with(|| record.clone());

        if entry.power_w.is_none() {
            entry.power_w = record.power_w;
        }
        if entry.voltage_v.is_none() {
            entry.voltage_v = record.voltage_v;
        }
        if entry.efficiency_pct.is_none() {
            entry.efficiency_pct = record.efficiency_pct;
        }
        if entry.lambda_nm.is_none() {
            entry.lambda_nm = record.lambda_nm;
        }
        if entry.shift_nm.is_none() {
            entry.shift_nm = record.shift_nm;
        }
        if entry.wavelength_2a_nm.is_none() {
            entry.wavelength_2a_nm = record.wavelength_2a_nm;
        }
        if entry.wavelength_cold_nm.is_none() {
            entry.wavelength_cold_nm = record.wavelength_cold_nm;
        }
    }

    merged.into_values().collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn build_record(
        id: &str,
        warehouse: &str,
        wavelength: f64,
        owner: &str,
        box_num: &str,
    ) -> CosRecord {
        CosRecord {
            device_id: id.to_string(),
            warehouse: Some(warehouse.to_string()),
            isolation: Some(String::from("否")),
            item_num: Some(String::from("A")),
            box_num: Some(box_num.to_string()),
            owner: Some(owner.to_string()),
            cold_wavelength_nm: Some(wavelength - 1.0),
            center_wavelength_nm: Some(wavelength - 0.5),
            two_a_wavelength_nm: Some(wavelength),
            peak_wavelength_nm: None,
        }
    }

    fn default_grouping_req() -> CosGroupingRequest {
        CosGroupingRequest {
            wavelength_field: WavelengthField::TwoA,
            group_size: 4,
            max_diff_nm: 0.2,
            avg_min_nm: None,
            avg_max_nm: None,
            strict_mode: None,
            uniformity_cv: None,
            low_min_nm: None,
            low_max_nm: None,
            high_min_nm: None,
            high_max_nm: None,
        }
    }

    #[test]
    fn test_step1_filters_expected_records() {
        let records = vec![
            build_record("d1", "良品仓", 965.0, "模块事业部", "b1"),
            build_record("d2", "研发工程仓", 966.0, "模块事业部", "b2"),
            build_record("d3", "其他仓", 965.5, "模块事业部", "b3"),
        ];

        let req = CosStep1Request {
            wavelength_field: WavelengthField::TwoA,
            wavelength_min_nm: 964.9,
            wavelength_max_nm: 965.6,
        };

        let filtered = filter_cos_step1(&records, &req);
        assert_eq!(filtered.len(), 1);
        assert_eq!(filtered[0].device_id, "d1");
    }

    #[test]
    fn test_step1_keeps_usable_owners_and_duplicate_device_ids() {
        let records = vec![
            build_record("d1", "良品仓", 965.0, "模块事业部", "b1"),
            build_record("d2", "良品仓", 965.1, "长光华芯", "b2"),
            build_record("d1", "研发工程仓", 965.2, "其他", "b3"),
            build_record("d3", "报废1仓", 965.3, "其他", "b4"),
        ];

        let req = CosStep1Request {
            wavelength_field: WavelengthField::TwoA,
            wavelength_min_nm: 964.9,
            wavelength_max_nm: 965.4,
        };

        let filtered = filter_cos_step1(&records, &req);
        let ids: Vec<&str> = filtered.iter().map(|record| record.device_id.as_str()).collect();

        assert_eq!(ids, vec!["d1", "d2", "d1", "d3"]);
    }

    #[test]
    fn test_step2_prioritizes_available_warehouse() {
        let records = vec![
            build_record("d1", "良品仓", 965.0, "模块事业部", "b1"),
            build_record("d2", "研发工程仓", 965.1, "模块事业部", "b2"),
            build_record("d3", "良品仓", 965.2, "长光华芯", "b1"),
            build_record("d4", "生产验证仓", 965.3, "其他", "b3"),
        ];

        let req = CosStep2Request {
            wavelength_field: WavelengthField::TwoA,
            wavelength_min_nm: 964.9,
            wavelength_max_nm: 965.4,
            required_count: 2,
            item_num_filter: None,
            box_num_filter: None,
        };

        let selected = filter_cos_step2(&records, &req);
        assert_eq!(selected.len(), 2);
        assert!(
            selected
                .iter()
                .all(|record| record.warehouse.as_deref() == Some("良品仓"))
        );
    }

    #[test]
    fn test_step2_does_not_fallback_to_other_warehouses() {
        let records = vec![
            build_record("d1", "其他仓", 965.0, "模块事业部", "b1"),
            build_record("d2", "其他仓", 965.1, "长光华芯", "b2"),
        ];

        let req = CosStep2Request {
            wavelength_field: WavelengthField::TwoA,
            wavelength_min_nm: 964.9,
            wavelength_max_nm: 965.2,
            required_count: 2,
            item_num_filter: None,
            box_num_filter: None,
        };

        let selected = filter_cos_step2(&records, &req);
        assert!(selected.is_empty());
    }

    #[test]
    fn test_step2_falls_back_when_wavelength_missing() {
        let records = vec![
            CosRecord {
                device_id: String::from("d1"),
                warehouse: Some(String::from("良品仓")),
                isolation: Some(String::from("否")),
                item_num: Some(String::from("A")),
                box_num: Some(String::from("b1")),
                owner: Some(String::from("模块事业部")),
                cold_wavelength_nm: None,
                center_wavelength_nm: None,
                two_a_wavelength_nm: None,
                peak_wavelength_nm: None,
            },
            CosRecord {
                device_id: String::from("d2"),
                warehouse: Some(String::from("良品仓")),
                isolation: Some(String::from("否")),
                item_num: Some(String::from("A")),
                box_num: Some(String::from("b2")),
                owner: Some(String::from("长光华芯")),
                cold_wavelength_nm: None,
                center_wavelength_nm: None,
                two_a_wavelength_nm: None,
                peak_wavelength_nm: None,
            },
        ];

        let req = CosStep2Request {
            wavelength_field: WavelengthField::TwoA,
            wavelength_min_nm: 964.9,
            wavelength_max_nm: 965.2,
            required_count: 1,
            item_num_filter: None,
            box_num_filter: None,
        };

        let selected = filter_cos_step2(&records, &req);
        assert_eq!(selected.len(), 1);
        assert_eq!(selected[0].device_id, "d1");
    }

    #[test]
    fn test_group_greedy_returns_expected_groups() {
        let records = (0..8)
            .map(|index| {
                build_record(
                    &format!("d{index}"),
                    "良品仓",
                    965.0 + index as f64 * 0.05,
                    "模块事业部",
                    "b1",
                )
            })
            .collect::<Vec<_>>();

        let req = default_grouping_req();

        let grouped = group_wavelengths_greedy(&records, &req).expect("grouping should succeed");
        assert_eq!(grouped.groups.len(), 2);
        assert_eq!(grouped.remaining.len(), 0);
    }

    #[test]
    fn test_group_optimal_returns_expected_groups() {
        let records = (0..12)
            .map(|index| {
                build_record(
                    &format!("d{index}"),
                    "良品仓",
                    965.0 + index as f64 * 0.05,
                    "模块事业部",
                    "b1",
                )
            })
            .collect::<Vec<_>>();

        let req = default_grouping_req();
        let grouped =
            group_wavelengths_optimal(&records, &req).expect("optimal grouping should succeed");
        assert_eq!(grouped.groups.len(), 3);
        assert_eq!(grouped.remaining.len(), 0);
    }

    #[test]
    fn test_group_flat_top_returns_groups() {
        let records = (0..10)
            .map(|index| {
                build_record(
                    &format!("d{index}"),
                    "良品仓",
                    965.0 + index as f64 * 0.05,
                    "模块事业部",
                    "b1",
                )
            })
            .collect::<Vec<_>>();

        let mut req = default_grouping_req();
        req.group_size = 5;
        req.max_diff_nm = 0.5;
        let grouped =
            group_wavelengths_flat_top(&records, &req).expect("flat-top grouping should succeed");
        assert!(!grouped.groups.is_empty());
        assert!(grouped.groups.iter().all(|group| group.len() == 5));
    }

    #[test]
    fn test_group_huang_meng_returns_groups() {
        let mut records = Vec::new();
        for index in 0..6 {
            records.push(build_record(
                &format!("low-{index}"),
                "良品仓",
                963.2 + index as f64 * 0.04,
                "模块事业部",
                "b1",
            ));
        }
        for index in 0..6 {
            records.push(build_record(
                &format!("high-{index}"),
                "良品仓",
                965.0 + index as f64 * 0.03,
                "模块事业部",
                "b2",
            ));
        }

        let mut req = default_grouping_req();
        req.group_size = 4;
        req.max_diff_nm = 2.5;
        req.avg_min_nm = Some(964.0);
        req.avg_max_nm = Some(964.3);
        req.low_min_nm = Some(963.2);
        req.low_max_nm = Some(963.5);
        req.high_min_nm = Some(965.0);
        req.high_max_nm = Some(965.3);

        let grouped = group_wavelengths_huang_meng(&records, &req)
            .expect("huang-meng grouping should succeed");
        assert!(!grouped.groups.is_empty());
        assert!(grouped.groups.iter().all(|group| group.len() == 4));
    }

    #[test]
    fn test_merge_data_fetch_records() {
        let first = DataFetchRecord {
            entry_id: String::from("shell-a"),
            test_category: String::from("耦合测试"),
            current_a: Some(2.0),
            power_w: Some(7.1),
            voltage_v: Some(1.8),
            efficiency_pct: Some(55.0),
            lambda_nm: None,
            shift_nm: None,
            wavelength_2a_nm: None,
            wavelength_cold_nm: None,
        };
        let second = DataFetchRecord {
            entry_id: String::from("shell-a"),
            test_category: String::from("耦合测试"),
            current_a: Some(2.0),
            power_w: None,
            voltage_v: None,
            efficiency_pct: None,
            lambda_nm: Some(965.2),
            shift_nm: Some(0.2),
            wavelength_2a_nm: Some(965.2),
            wavelength_cold_nm: Some(964.8),
        };

        let merged = merge_data_fetch_records(&[first, second]);
        assert_eq!(merged.len(), 1);
        assert_eq!(merged[0].power_w, Some(7.1));
        assert_eq!(merged[0].lambda_nm, Some(965.2));
    }

    #[test]
    fn test_merge_data_fetch_records_skips_missing_current() {
        let row = DataFetchRecord {
            entry_id: String::from("shell-a"),
            test_category: String::from("耦合测试"),
            current_a: None,
            power_w: Some(3.4),
            voltage_v: Some(1.8),
            efficiency_pct: Some(0.41),
            lambda_nm: None,
            shift_nm: None,
            wavelength_2a_nm: None,
            wavelength_cold_nm: None,
        };

        let merged = merge_data_fetch_records(&[row]);
        assert!(merged.is_empty());
    }
}
