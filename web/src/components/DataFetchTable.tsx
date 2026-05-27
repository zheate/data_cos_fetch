import { useState, useMemo, useDeferredValue } from 'react';
import type { DataFetchRow } from '../helpers/types';
import { toFixed } from '../helpers/utils';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { TablePager } from '@/components/TablePager';

const COLUMNS = [
  { key: 'entry_id', label: '条目' },
  { key: 'test_category', label: '测试类别' },
  { key: 'current_a', label: '电流' },
  { key: 'power_w', label: '功率' },
  { key: 'voltage_v', label: '电压' },
  { key: 'efficiency_pct', label: '效率%' },
  { key: 'lambda_nm', label: '波长' },
  { key: 'shift_nm', label: '波长偏移' },
  { key: 'wavelength_2a_nm', label: '2A' },
  { key: 'wavelength_cold_nm', label: '冷波长' },
];

const PAGE_SIZE = 50;

import { Input } from '@/components/ui/input';
import { Search, Loader2 } from 'lucide-react';

export function DataFetchTable({ rows }: { rows: DataFetchRow[] }) {
  const [page, setPage] = useState(1);
  const [searchTerm, setSearchTerm] = useState('');
  const deferredSearchTerm = useDeferredValue(searchTerm);

  const filteredRows = useMemo(() => {
    if (!deferredSearchTerm.trim()) return rows;
    const lower = deferredSearchTerm.toLowerCase();
    return rows.filter((row) =>
      row.entry_id.toLowerCase().includes(lower) ||
      (row.test_category && row.test_category.toLowerCase().includes(lower))
    );
  }, [rows, deferredSearchTerm]);

  // Reset page when search term changes
  useMemo(() => {
    setPage(1);
  }, [deferredSearchTerm]);

  const totalPages = Math.max(1, Math.ceil(filteredRows.length / PAGE_SIZE));

  const pageRows = useMemo(
    () => filteredRows.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE),
    [filteredRows, page],
  );

  if (rows.length === 0) {
    return <p className="py-6 text-center text-sm text-muted-foreground">暂无数据</p>;
  }

  const renderCell = (row: DataFetchRow, key: string) => {
    switch (key) {
      case 'entry_id':
        return row.entry_id;
      case 'test_category':
        return row.test_category;
      case 'current_a':
        return toFixed(row.current_a, 3);
      case 'power_w':
        return toFixed(row.power_w, 3);
      case 'voltage_v':
        return toFixed(row.voltage_v, 3);
      case 'efficiency_pct':
        return toFixed(row.efficiency_pct, 3);
      case 'lambda_nm':
        return toFixed(row.lambda_nm, 3);
      case 'shift_nm':
        return toFixed(row.shift_nm, 3);
      case 'wavelength_2a_nm':
        return toFixed(row.wavelength_2a_nm, 3);
      case 'wavelength_cold_nm':
        return toFixed(row.wavelength_cold_nm, 3);
      default:
        return '-';
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 max-w-sm">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            type="search"
            placeholder="搜索条目或测试类别..."
            className="pl-9 h-9"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
        {searchTerm !== deferredSearchTerm && (
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        )}
      </div>
      <div className="overflow-hidden rounded-xl border bg-card">
        <div className="max-h-[460px] overflow-auto">
          <Table aria-label="Data fetch results">
            <TableHeader className="sticky top-0 z-10 bg-muted/80 backdrop-blur">
              <TableRow>
                {COLUMNS.map((column) => (
                  <TableHead
                    key={column.key}
                    className={column.key === 'entry_id' || column.key === 'test_category' ? undefined : 'text-right'}
                  >
                    {column.label}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {pageRows.map((row, index) => (
                <TableRow key={`${row.entry_id}-${row.test_category}-${(page - 1) * PAGE_SIZE + index}`}>
                  {COLUMNS.map((column) => (
                    <TableCell
                      key={column.key}
                      className={column.key === 'entry_id' || column.key === 'test_category' ? undefined : 'text-right tabular-nums'}
                    >
                      {renderCell(row, column.key)}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
        <TablePager page={page} totalPages={totalPages} totalRows={filteredRows.length} onPageChange={setPage} />
      </div>
    </div>
  );
}
