import { useDeferredValue, useMemo, useState } from 'react';
import { Loader2, Search } from 'lucide-react';
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
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { TablePager } from '@/components/TablePager';
import { cn } from '@/lib/utils';

type ColumnKey = keyof DataFetchRow;

const COLUMNS: Array<{
  key: ColumnKey;
  label: string;
  align?: 'left' | 'right';
  className?: string;
}> = [
  { key: 'entry_id', label: '条目', className: 'min-w-[220px]' },
  { key: 'test_category', label: '测试类别', className: 'min-w-[120px]' },
  { key: 'current_a', label: '电流 A', align: 'right' },
  { key: 'power_w', label: '功率 W', align: 'right' },
  { key: 'voltage_v', label: '电压 V', align: 'right' },
  { key: 'efficiency_pct', label: '效率 %', align: 'right' },
  { key: 'lambda_nm', label: '波长 nm', align: 'right' },
  { key: 'shift_nm', label: '偏移 nm', align: 'right' },
  { key: 'wavelength_2a_nm', label: '2A nm', align: 'right' },
  { key: 'wavelength_cold_nm', label: '冷波长 nm', align: 'right' },
];

const PAGE_SIZE = 50;

export function DataFetchTable({ rows }: { rows: DataFetchRow[] }) {
  const [page, setPage] = useState(1);
  const [searchTerm, setSearchTerm] = useState('');
  const deferredSearchTerm = useDeferredValue(searchTerm);
  const hasPendingSearch = searchTerm !== deferredSearchTerm;

  const filteredRows = useMemo(() => {
    if (!deferredSearchTerm.trim()) return rows;
    const lower = deferredSearchTerm.toLowerCase();
    return rows.filter((row) =>
      row.entry_id.toLowerCase().includes(lower) ||
      (row.test_category && row.test_category.toLowerCase().includes(lower))
    );
  }, [rows, deferredSearchTerm]);

  const totalPages = Math.max(1, Math.ceil(filteredRows.length / PAGE_SIZE));
  const normalizedPage = Math.min(page, totalPages);

  const pageRows = useMemo(
    () => filteredRows.slice((normalizedPage - 1) * PAGE_SIZE, normalizedPage * PAGE_SIZE),
    [filteredRows, normalizedPage],
  );

  const visibleStart = filteredRows.length === 0 ? 0 : (normalizedPage - 1) * PAGE_SIZE + 1;
  const visibleEnd = Math.min(normalizedPage * PAGE_SIZE, filteredRows.length);

  const renderCell = (row: DataFetchRow, key: ColumnKey) => {
    switch (key) {
      case 'entry_id':
        return (
          <span className="block max-w-[260px] truncate font-medium text-foreground" title={row.entry_id}>
            {row.entry_id}
          </span>
        );
      case 'test_category':
        return row.test_category || '-';
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

  if (rows.length === 0) {
    return (
      <div className="flex min-h-[280px] items-center justify-center rounded-lg border border-dashed bg-muted/10 px-6 text-center">
        <div>
          <p className="text-sm font-semibold text-foreground">没有提取到记录</p>
          <p className="mt-1 text-xs text-muted-foreground">错误和提示会显示在结果区上方。</p>
        </div>
      </div>
    );
  }

  return (
    <div className="result-table-shell flex flex-col gap-3">
      <div className="flex flex-col gap-3 rounded-lg border bg-background/65 px-3.5 py-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-foreground">明细</h3>
            <Badge variant="secondary" className="rounded-md">
              {filteredRows.length} / {rows.length}
            </Badge>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            显示 {visibleStart}-{visibleEnd}，每页 {PAGE_SIZE} 条
          </p>
        </div>

        <div className="flex w-full items-center gap-2 sm:w-[320px]">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              placeholder="搜索条目或测试类别"
              className="h-9 pl-9"
              value={searchTerm}
              onChange={(event) => {
                setSearchTerm(event.target.value);
                setPage(1);
              }}
            />
          </div>
          {hasPendingSearch && (
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          )}
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border bg-card shadow-sm">
        <div className="max-h-[590px] overflow-auto">
          <Table aria-label="数据提取结果">
            <TableHeader className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
              <TableRow>
                {COLUMNS.map((column) => (
                  <TableHead
                    key={column.key}
                    className={cn(
                      'h-9 whitespace-nowrap text-xs',
                      column.align === 'right' && 'text-right',
                      column.className,
                    )}
                  >
                    {column.label}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {pageRows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={COLUMNS.length} className="h-32 text-center text-sm text-muted-foreground">
                    没有匹配的记录
                  </TableCell>
                </TableRow>
              ) : (
                pageRows.map((row, index) => (
                  <TableRow
                    key={`${row.entry_id}-${row.test_category}-${(normalizedPage - 1) * PAGE_SIZE + index}`}
                    className="hover:bg-muted/35"
                  >
                    {COLUMNS.map((column) => (
                      <TableCell
                        key={column.key}
                        className={cn(
                          'whitespace-nowrap text-xs',
                          column.align === 'right' && 'text-right font-mono tabular-nums',
                          column.className,
                        )}
                      >
                        {renderCell(row, column.key)}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
        <TablePager page={normalizedPage} totalPages={totalPages} totalRows={filteredRows.length} onPageChange={setPage} />
      </div>
    </div>
  );
}
