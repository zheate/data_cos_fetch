import type { CosRow, HistogramBin, WavelengthField } from './types';

// ── Constants ──────────────────────────────────────────────────────

export const DEFAULT_TEST_CATEGORIES = [
  '耦合测试',
  'Pre测试',
  '低温储存后测试',
  'Post测试',
  '封盖测试',
  '高温测试',
];
export const MEASUREMENTS = ['LVI', 'Rth', 'lambd'];
export const WAREHOUSE_AVAILABLE = ['良品仓'];
export const WAREHOUSE_NEED_CONFIRM = ['研发工程仓', '生产验证仓', '报废1仓'];

export const WAVELENGTH_LABEL_TO_FIELD: Record<string, WavelengthField> = {
  '2A波长': 'two_a',
  冷波长: 'cold',
  中心波长: 'center',
  峰值波长: 'peak',
};

export const WAVELENGTH_FIELD_TO_LABEL: Record<WavelengthField, string> = {
  two_a: '2A波长',
  cold: '冷波长',
  center: '中心波长',
  peak: '峰值波长',
};

// ── Parsing helpers ────────────────────────────────────────────────

export const parseLines = (raw: string): string[] =>
  raw
    .replaceAll('，', '\n')
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0);

export const parseCurrentPoints = (raw: string): number[] | null => {
  const text = raw.trim();
  if (text.toLowerCase() === 'a') {
    return null;
  }

  const currents: number[] = [];
  const lines = raw.replaceAll('，', ',').replaceAll('～', '~').split('\n');

  for (const line of lines) {
    for (const pieceRaw of line.split(',')) {
      const piece = pieceRaw.trim().replaceAll('～', '~');
      if (piece.length === 0) {
        continue;
      }

      if (!piece.includes('~') && piece.slice(1).indexOf('-') === -1) {
        const tokens = piece.split(/\s+/).filter((token) => token.length > 0);
        if (tokens.length > 1) {
          for (const token of tokens) {
            const value = Number.parseFloat(token);
            if (!Number.isFinite(value)) {
              throw new Error(`Cannot parse current value: ${pieceRaw}`);
            }
            currents.push(value);
          }
          continue;
        }
      }

      let range: [string, string] | null = null;
      if (piece.includes('~')) {
        const parts = piece.split('~', 2);
        range = [parts[0], parts[1]];
      } else {
        const dashIndex = piece.slice(1).indexOf('-');
        if (dashIndex !== -1) {
          const index = dashIndex + 1;
          range = [piece.slice(0, index), piece.slice(index + 1)];
        }
      }

      if (range) {
        const start = Number.parseFloat(range[0].trim());
        const end = Number.parseFloat(range[1].trim());
        if (!Number.isFinite(start) || !Number.isFinite(end)) {
          throw new Error(`Cannot parse current range: ${pieceRaw}`);
        }

        if (Number.isInteger(start) && Number.isInteger(end)) {
          const step = end >= start ? 1 : -1;
          for (let value = start; step > 0 ? value <= end : value >= end; value += step) {
            currents.push(value);
          }
        } else {
          currents.push(start, end);
        }
        continue;
      }

      const value = Number.parseFloat(piece);
      if (!Number.isFinite(value)) {
        throw new Error(`Cannot parse current value: ${pieceRaw}`);
      }
      currents.push(value);
    }
  }

  return currents;
};

export const parseOptionalNumber = (raw: string): number | undefined => {
  const trimmed = raw.trim();
  if (trimmed.length === 0) {
    return undefined;
  }
  const parsed = Number.parseFloat(trimmed);
  return Number.isFinite(parsed) ? parsed : undefined;
};

// ── Formatting ─────────────────────────────────────────────────────

export function toFixed(value: number | null | undefined, digits: number): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-';
  }
  return value.toFixed(digits);
}

export const formatSize = (sizeBytes: number): string => {
  const sizeMb = sizeBytes / (1024 * 1024);
  return `${sizeMb.toFixed(1)}MB`;
};

export const formatEpoch = (epochSeconds: number): string => {
  if (!Number.isFinite(epochSeconds) || epochSeconds <= 0) {
    return '-';
  }
  const date = new Date(epochSeconds * 1000);
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

// ── Math / statistics ──────────────────────────────────────────────

export function standardDeviation(values: number[]): number {
  if (values.length <= 1) {
    return 0;
  }
  const mean = values.reduce((acc, value) => acc + value, 0) / values.length;
  const variance = values.reduce((acc, value) => acc + (value - mean) ** 2, 0) / values.length;
  return Math.sqrt(variance);
}

export const buildHistogram = (values: number[], bins: number): HistogramBin[] => {
  if (values.length === 0 || bins <= 0) {
    return [];
  }

  const min = Math.min(...values);
  const max = Math.max(...values);
  if (Math.abs(max - min) < Number.EPSILON) {
    return [{ start: min, end: max, count: values.length }];
  }

  const width = (max - min) / bins;
  const counts = new Array<number>(bins).fill(0);
  for (const value of values) {
    const index = Math.min(bins - 1, Math.max(0, Math.floor((value - min) / width)));
    counts[index] += 1;
  }

  return counts
    .map((count, index) => {
      if (count === 0) {
        return null;
      }
      const start = min + index * width;
      const end = index === bins - 1 ? max : start + width;
      return { start, end, count };
    })
    .filter((bin): bin is HistogramBin => bin !== null);
};

// ── CosRow helpers ─────────────────────────────────────────────────

export const getWavelengthValue = (row: CosRow, field: WavelengthField): number | null => {
  if (field === 'two_a') {
    return typeof row.two_a_wavelength_nm === 'number' ? row.two_a_wavelength_nm : null;
  }
  if (field === 'cold') {
    return typeof row.cold_wavelength_nm === 'number' ? row.cold_wavelength_nm : null;
  }
  if (field === 'peak') {
    return typeof row.peak_wavelength_nm === 'number' ? row.peak_wavelength_nm : null;
  }
  return typeof row.center_wavelength_nm === 'number' ? row.center_wavelength_nm : null;
};

export const dedupeByDeviceId = (rows: CosRow[]): { rows: CosRow[]; removed: number } => {
  const seen = new Set<string>();
  const deduped: CosRow[] = [];
  let removed = 0;

  for (const row of rows) {
    const deviceId = (row.device_id ?? '').trim();
    if (deviceId.length === 0) {
      continue;
    }
    if (seen.has(deviceId)) {
      removed += 1;
      continue;
    }
    seen.add(deviceId);
    deduped.push(row);
  }

  return { rows: deduped, removed };
};
