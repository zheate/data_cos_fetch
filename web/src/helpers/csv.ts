import type { CosGroupResponse, CosRow } from './types';
import * as XLSX from 'xlsx';

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

export const downloadRowsAsXlsx = (rows: CosRow[], filename: string) => {
  if (rows.length === 0) return;
  const worksheet = XLSX.utils.json_to_sheet(rows, { header: COS_ROW_HEADERS });
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, 'Data');
  XLSX.writeFile(workbook, filename);
};

export const downloadGroupingAsXlsx = (result: CosGroupResponse, filename: string) => {
  if (result.group_count === 0 && result.remaining_count === 0) return;
  const rows: any[] = [];
  result.groups.forEach((group, index) => {
    const groupId = `Group_${String(index + 1).padStart(3, '0')}`;
    group.forEach((row) => {
      rows.push({ group_id: groupId, ...row });
    });
  });
  result.remaining.forEach((row) => {
    rows.push({ group_id: 'REMAINING', ...row });
  });
  const worksheet = XLSX.utils.json_to_sheet(rows, { header: ['group_id', ...COS_ROW_HEADERS] });
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, 'Grouped Data');
  XLSX.writeFile(workbook, filename);
};

export const downloadDataFetchAsXlsx = (rows: unknown[], filename: string) => {
  if (rows.length === 0) return;
  const headerSet = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row as Record<string, unknown>)) {
      headerSet.add(key);
    }
  }
  let headers = Array.from(headerSet);
  const priorityKeys = ['entry_id', 'test_category'];
  const sortedHeaders = [];
  for (const p of priorityKeys) {
    if (headers.includes(p)) sortedHeaders.push(p);
  }
  headers = [...sortedHeaders, ...headers.filter(h => !priorityKeys.includes(h))];

  const worksheet = XLSX.utils.json_to_sheet(rows as any[], { header: headers });
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, 'Data Fetch Output');
  XLSX.writeFile(workbook, filename);
};
