import type { CosGroupResponse, CosRow } from './types';

const COS_ROW_HEADERS = [
  'device_id',
  'warehouse',
  'isolation',
  'item_num',
  'box_num',
  'owner',
  'cold_wavelength_nm',
  'center_wavelength_nm',
  'two_a_wavelength_nm',
  'peak_wavelength_nm',
];

const csvEscape = (value: unknown): string => {
  if (value === null || value === undefined) {
    return '';
  }
  const text = String(value);
  if (text.includes(',') || text.includes('"') || text.includes('\n')) {
    return `"${text.replaceAll('"', '""')}"`;
  }
  return text;
};

const rowToCsvValues = (row: CosRow): string[] => {
  const record = row as Record<string, unknown>;
  return COS_ROW_HEADERS.map((header) => csvEscape(record[header]));
};

const triggerCsvDownload = (lines: string[], filename: string) => {
  const blob = new Blob([`\uFEFF${lines.join('\n')}`], { type: 'text/csv;charset=utf-8;' });
  const link = document.createElement('a');
  link.href = URL.createObjectURL(blob);
  link.download = filename;
  link.click();
  URL.revokeObjectURL(link.href);
};

export const downloadRowsAsCsv = (rows: CosRow[], filename: string) => {
  if (rows.length === 0) {
    return;
  }
  const lines = [COS_ROW_HEADERS.join(',')];
  for (const row of rows) {
    lines.push(rowToCsvValues(row).join(','));
  }
  triggerCsvDownload(lines, filename);
};

export const downloadGroupingAsCsv = (result: CosGroupResponse, filename: string) => {
  if (result.group_count === 0 && result.remaining_count === 0) {
    return;
  }

  const lines = [['group_id', ...COS_ROW_HEADERS].join(',')];
  result.groups.forEach((group, index) => {
    const groupId = `Group_${String(index + 1).padStart(3, '0')}`;
    group.forEach((row) => {
      lines.push([csvEscape(groupId), ...rowToCsvValues(row)].join(','));
    });
  });
  result.remaining.forEach((row) => {
    lines.push([csvEscape('REMAINING'), ...rowToCsvValues(row)].join(','));
  });
  triggerCsvDownload(lines, filename);
};

export const downloadDataFetchAsCsv = (rows: unknown[], filename: string) => {
  if (rows.length === 0) return;
  const headerSet = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row as Record<string, unknown>)) {
      headerSet.add(key);
    }
  }
  // To ensure entry_id and test_category (if exist) are at the front
  let headers = Array.from(headerSet);
  const priorityKeys = ['entry_id', 'test_category'];
  const sortedHeaders = [];
  for (const p of priorityKeys) {
    if (headers.includes(p)) sortedHeaders.push(p);
  }
  headers = [...sortedHeaders, ...headers.filter(h => !priorityKeys.includes(h))];

  const lines = [headers.map(csvEscape).join(',')];
  for (const row of rows) {
    const record = row as Record<string, unknown>;
    lines.push(headers.map((h) => csvEscape(record[h])).join(','));
  }
  triggerCsvDownload(lines, filename);
};
