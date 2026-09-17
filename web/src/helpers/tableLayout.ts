/**
 * Table layout and column width calculation utilities.
 * Automatically computes optimal column widths based on header text and visible/result data.
 */

export interface ColumnWidthSpec<T> {
  key: keyof T;
  label: string;
  minWidth?: number;
  maxWidth?: number;
  isMono?: boolean;
  align?: 'left' | 'center' | 'right';
  /** Extra width buffer in px, e.g. for icons or tooltips */
  extraPadding?: number;
  /** Custom text extractor for row value */
  getText?: (row: T) => string;
}

/**
 * Estimate pixel width of a string rendered in text-xs (12px) font.
 * Based on typical CJK, latin, mono digits, and symbol font metrics.
 */
export function estimateTextWidth(text: string, isMono = false): number {
  if (!text) return 0;
  let width = 0;
  for (let i = 0; i < text.length; i++) {
    const code = text.charCodeAt(i);
    if (code >= 0x4e00 && code <= 0x9fff) {
      // CJK / full-width Chinese characters
      width += 13.5;
    } else if (isMono) {
      // Monospace digits and characters
      width += 7.8;
    } else if (code >= 65 && code <= 90) {
      // A-Z uppercase latin
      width += 8.2;
    } else if (code >= 48 && code <= 57) {
      // 0-9 digits
      width += 7.5;
    } else if (code === 32) {
      // Space
      width += 4.5;
    } else if (code === 46 || code === 44 || code === 58) {
      // . , :
      width += 4;
    } else {
      // Lowercase latin, symbols (- / %)
      width += 6.8;
    }
  }
  return Math.ceil(width);
}

/**
 * Compute optimal column widths from result data.
 * Checks header label length and scans rows to find max content width.
 */
export function computeColumnWidths<T>(
  columns: ColumnWidthSpec<T>[],
  rows: T[],
  options?: {
    /** Max rows to sample for performance. Defaults to 200. */
    sampleLimit?: number;
    /** Base horizontal padding per cell in px. Defaults to 28 (px-3.5: 14px * 2). */
    basePadding?: number;
    /** Extra space for sort indicator arrow. Defaults to 16. */
    sortIndicatorWidth?: number;
  }
): Record<string, number> {
  const sampleLimit = options?.sampleLimit ?? 200;
  const basePadding = options?.basePadding ?? 28;
  const sortIndicatorWidth = options?.sortIndicatorWidth ?? 16;
  const sampleRows = rows.length > sampleLimit ? rows.slice(0, sampleLimit) : rows;

  const result: Record<string, number> = {};

  for (const col of columns) {
    const headerWidth = estimateTextWidth(col.label, false) + (col.extraPadding ?? 0) + sortIndicatorWidth;
    let maxCellWidth = 0;

    for (const row of sampleRows) {
      let cellText = '';
      if (col.getText) {
        cellText = col.getText(row);
      } else {
        const val = row[col.key];
        cellText = val !== null && val !== undefined ? String(val) : '';
      }

      if (cellText) {
        const w = estimateTextWidth(cellText, col.isMono ?? false);
        if (w > maxCellWidth) {
          maxCellWidth = w;
        }
      }
    }

    const contentWidth = Math.max(headerWidth, maxCellWidth);
    let calculated = contentWidth + basePadding;

    // Apply min/max constraints
    const minW = col.minWidth ?? 64;
    const maxW = col.maxWidth ?? 360;
    calculated = Math.max(minW, Math.min(maxW, calculated));

    result[String(col.key)] = Math.round(calculated);
  }

  return result;
}

// ── Sorting Helpers ────────────────────────────────────────────────

export type SortDirection = 'asc' | 'desc';

export interface SortState<K = string> {
  key: K;
  direction: SortDirection;
}

/**
 * Transition sort state on header click:
 * - Click different column -> 'asc'
 * - Click same column in 'asc' -> 'desc'
 * - Click same column in 'desc' -> null (reset to original order)
 */
export function getNextSortState<K>(
  current: SortState<K> | null,
  newKey: K
): SortState<K> | null {
  if (!current || current.key !== newKey) {
    return { key: newKey, direction: 'asc' };
  }
  if (current.direction === 'asc') {
    return { key: newKey, direction: 'desc' };
  }
  return null;
}

/**
 * Sort rows by key and direction.
 * Safely handles nulls/undefined, numbers, and strings with natural alphanumeric sorting.
 */
export function sortRows<T, K extends keyof T>(
  rows: T[],
  sortState: SortState<K> | null,
  getCustomValue?: (row: T, key: K) => unknown
): T[] {
  if (!sortState) return rows;
  const { key, direction } = sortState;
  const factor = direction === 'asc' ? 1 : -1;

  return [...rows].sort((a, b) => {
    const va = getCustomValue ? getCustomValue(a, key) : a[key];
    const vb = getCustomValue ? getCustomValue(b, key) : b[key];

    if (va === vb) return 0;
    if (va === null || va === undefined || va === '') return 1;
    if (vb === null || vb === undefined || vb === '') return -1;

    if (typeof va === 'number' && typeof vb === 'number') {
      return (va - vb) * factor;
    }

    const sa = String(va);
    const sb = String(vb);
    return sa.localeCompare(sb, undefined, { numeric: true, sensitivity: 'base' }) * factor;
  });
}

