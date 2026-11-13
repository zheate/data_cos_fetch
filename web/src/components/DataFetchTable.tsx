import { useDeferredValue, useMemo, useState } from 'react';
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react';
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
import { Button } from '@/components/ui/button';
import { TablePager } from '@/components/TablePager';
import { cn } from '@/lib/utils';
import {
  computeColumnWidths,
  getNextSortState,
  sortRows,
  type ColumnWidthSpec,
  type SortState,
} from '../helpers/tableLayout';

type ColumnKey = keyof DataFetchRow;

type DataFetchTableProps = {
  rows: DataFetchRow[];
  searchTerm?: string;
  onSearchChange?: (value: string) => void;
  pageSize?: number;
};

function SortIndicator({ active, direction }: { active: boolean; direction?: 'asc' | 'desc' }) {
  if (active && direction === 'asc') return <ArrowUp className="size-3 shrink-0 text-primary" />;
  if (active && direction === 'desc') return <ArrowDown className="size-3 shrink-0 text-primary" />;
  return <ArrowUpDown className="size-3 shrink-0 text-muted-foreground/40 transition-colors group-hover:text-muted-foreground" />;
}

const COLUMNS: ColumnWidthSpec<DataFetchRow>[] = [
  { key: 'entry_id', label: '条目编号', align: 'left', minWidth: 150, maxWidth: 300, isMono: true, getText: (r) => r.entry_id },
  { key: 'test_category', label: '站别', align: 'left', minWidth: 96, maxWidth: 160, getText: (r) => r.test_category ?? '' },
  { key: 'current_a', label: '电流 A', align: 'right', minWidth: 80, isMono: true, getText: (r) => toFixed(r.current_a, 3) },
  { key: 'power_w', label: '功率 W', align: 'right', minWidth: 80, isMono: true, getText: (r) => toFixed(r.power_w, 3) },
  { key: 'voltage_v', label: '电压 V', align: 'right', minWidth: 80, isMono: true, getText: (r) => toFixed(r.voltage_v, 3) },
  { key: 'efficiency_pct', label: '效率 %', align: 'right', minWidth: 80, isMono: true, getText: (r) => toFixed(r.efficiency_pct, 3) },
  { key: 'lambda_nm', label: '波长 nm', align: 'right', minWidth: 90, isMono: true, getText: (r) => toFixed(r.lambda_nm, 3) },
  { key: 'shift_nm', label: '偏移 nm', align: 'right', minWidth: 84, isMono: true, getText: (r) => toFixed(r.shift_nm, 3) },
  { key: 'wavelength_2a_nm', label: '2A 波长 nm', align: 'right', minWidth: 104, isMono: true, getText: (r) => toFixed(r.wavelength_2a_nm, 3) },
  { key: 'wavelength_cold_nm', label: '冷波长 nm', align: 'right', minWidth: 100, isMono: true, getText: (r) => toFixed(r.wavelength_cold_nm, 3) },
];

export function DataFetchTable({
  rows,
  searchTerm = '',
  onSearchChange,
  pageSize = 50,
}: DataFetchTableProps) {
  const [page, setPage] = useState(1);
  const [sortState, setSortState] = useState<SortState<ColumnKey> | null>(null);
  const deferredSearchTerm = useDeferredValue(searchTerm);

  const filteredRows = useMemo(() => {
    if (!deferredSearchTerm.trim()) return rows;
    const lower = deferredSearchTerm.toLowerCase();
    return rows.filter((row) =>
      row.entry_id.toLowerCase().includes(lower) ||
      (row.test_category && row.test_category.toLowerCase().includes(lower)),
    );
  }, [rows, deferredSearchTerm]);

  const sortedRows = useMemo(() => sortRows(filteredRows, sortState), [filteredRows, sortState]);
  const columnWidths = useMemo(() => computeColumnWidths(COLUMNS, sortedRows, {
    basePadding: 24,
    sampleLimit: 300,
    sortIndicatorWidth: 16,
  }), [sortedRows]);

  const totalPages = Math.max(1, Math.ceil(sortedRows.length / pageSize));
  const normalizedPage = Math.min(page, totalPages);
  const pageRows = useMemo(
    () => sortedRows.slice((normalizedPage - 1) * pageSize, normalizedPage * pageSize),
    [sortedRows, normalizedPage, pageSize],
  );

  const handleSort = (key: ColumnKey) => {
    setSortState((prev) => getNextSortState(prev, key));
    setPage(1);
  };

  const renderCell = (row: DataFetchRow, key: ColumnKey) => {
    if (key === 'entry_id') {
      return <span className="block truncate font-mono text-xs font-medium text-foreground" title={row.entry_id}>{row.entry_id}</span>;
    }
    if (key === 'test_category') {
      return row.test_category
        ? <span className="block truncate text-xs text-muted-foreground" title={row.test_category}>{row.test_category}</span>
        : <span className="text-muted-foreground">-</span>;
    }
    const value = row[key];
    return toFixed(typeof value === 'number' ? value : null, 3);
  };

  if (rows.length === 0) {
    return (
      <div className="flex min-h-[360px] flex-col items-center justify-center px-6 py-12 text-center" role="status">
        <p className="text-sm font-medium text-foreground">没有提取到记录</p>
        <p className="mt-1.5 max-w-sm text-xs leading-5 text-muted-foreground">查看“异常记录”确认原因，或调整左侧条目、站别和电流点后重新提取。</p>
      </div>
    );
  }

  return (
    <div className="result-table-shell flex flex-col">
      <Table containerClassName="max-h-[min(66dvh,700px)] overflow-auto" aria-label="数据提取结果">
        <colgroup>
          {COLUMNS.map((column) => (
            <col key={String(column.key)} style={{ minWidth: `${columnWidths[String(column.key)]}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow>
            {COLUMNS.map((column) => {
              const isActive = sortState?.key === column.key;
              return (
                <TableHead
                  key={String(column.key)}
                  aria-sort={isActive ? (sortState.direction === 'asc' ? 'ascending' : 'descending') : 'none'}
                  className={cn(
                    'h-8 whitespace-nowrap bg-muted/45 px-3 text-[11px] font-normal text-muted-foreground',
                    column.align === 'right' && 'text-right',
                    isActive && 'font-medium text-foreground',
                  )}
                >
                  <button
                    type="button"
                    onClick={() => handleSort(column.key as ColumnKey)}
                    title={`按${column.label}排序`}
                    className={cn(
                      'group inline-flex h-7 w-full items-center gap-1 rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50',
                      column.align === 'right' && 'justify-end',
                    )}
                  >
                    <span>{column.label}</span>
                    <SortIndicator active={isActive} direction={isActive ? sortState.direction : undefined} />
                  </button>
                </TableHead>
              );
            })}
          </TableRow>
        </TableHeader>
        <TableBody>
          {pageRows.length === 0 ? (
            <TableRow>
              <TableCell colSpan={COLUMNS.length} className="h-52 text-center">
                <p className="text-sm font-medium text-foreground">没有匹配的记录</p>
                <p className="mt-1 text-xs text-muted-foreground">尝试其他条目编号或站别。</p>
                {onSearchChange && (
                  <Button type="button" variant="outline" size="sm" className="mt-4" onClick={() => { onSearchChange(''); setPage(1); }}>
                    清除搜索
                  </Button>
                )}
              </TableCell>
            </TableRow>
          ) : pageRows.map((row, index) => (
            <TableRow key={`${row.entry_id}-${row.test_category}-${(normalizedPage - 1) * pageSize + index}`} className="hover:bg-muted/35">
              {COLUMNS.map((column) => (
                <TableCell
                  key={String(column.key)}
                  className={cn(
                    'h-9 whitespace-nowrap px-3 py-0 text-xs',
                    column.align === 'right' && 'font-mono tabular-nums text-foreground',
                  )}
                >
                  {renderCell(row, column.key as ColumnKey)}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p className="border-t px-3 py-1.5 text-[10px] text-muted-foreground md:hidden">左右滑动查看全部列</p>
      <TablePager
        page={normalizedPage}
        totalPages={totalPages}
        totalRows={sortedRows.length}
        pageSize={pageSize}
        onPageChange={setPage}
      />
    </div>
  );
}
