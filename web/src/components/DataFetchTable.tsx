import { useDeferredValue, useMemo, useState } from 'react';
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  Loader2,
  Maximize2,
  Minimize2,
  Search,
} from 'lucide-react';
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

function SortIndicator({
  active,
  direction,
}: {
  active: boolean;
  direction?: 'asc' | 'desc';
}) {
  if (active && direction === 'asc') {
    return <ArrowUp className="h-3 w-3 text-primary shrink-0" />;
  }
  if (active && direction === 'desc') {
    return <ArrowDown className="h-3 w-3 text-primary shrink-0" />;
  }
  return (
    <ArrowUpDown className="h-3 w-3 text-muted-foreground/30 transition-colors group-hover:text-muted-foreground shrink-0" />
  );
}

const COLUMNS: ColumnWidthSpec<DataFetchRow>[] = [
  { key: 'entry_id', label: '条目', align: 'left', minWidth: 125, maxWidth: 300, isMono: true, getText: (r) => r.entry_id },
  { key: 'test_category', label: '测试类别', align: 'center', minWidth: 78, maxWidth: 120, getText: (r) => r.test_category ?? '' },
  { key: 'current_a', label: '电流 A', align: 'right', minWidth: 68, isMono: true, getText: (r) => toFixed(r.current_a, 3) },
  { key: 'power_w', label: '功率 W', align: 'right', minWidth: 68, isMono: true, getText: (r) => toFixed(r.power_w, 3) },
  { key: 'voltage_v', label: '电压 V', align: 'right', minWidth: 65, isMono: true, getText: (r) => toFixed(r.voltage_v, 3) },
  { key: 'efficiency_pct', label: '效率 %', align: 'right', minWidth: 70, isMono: true, getText: (r) => toFixed(r.efficiency_pct, 3) },
  { key: 'lambda_nm', label: '波长 nm', align: 'right', minWidth: 76, isMono: true, getText: (r) => toFixed(r.lambda_nm, 3) },
  { key: 'shift_nm', label: '偏移 nm', align: 'right', minWidth: 70, isMono: true, getText: (r) => toFixed(r.shift_nm, 3) },
  { key: 'wavelength_2a_nm', label: '2A nm', align: 'right', minWidth: 76, isMono: true, getText: (r) => toFixed(r.wavelength_2a_nm, 3) },
  { key: 'wavelength_cold_nm', label: '冷波长 nm', align: 'right', minWidth: 84, isMono: true, getText: (r) => toFixed(r.wavelength_cold_nm, 3) },
];

const PAGE_SIZE = 50;

export function DataFetchTable({ rows }: { rows: DataFetchRow[] }) {
  const [page, setPage] = useState(1);
  const [searchTerm, setSearchTerm] = useState('');
  const [isCompact, setIsCompact] = useState(true);
  const [sortState, setSortState] = useState<SortState<ColumnKey> | null>(null);
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

  // Sort rows based on active header sort
  const sortedRows = useMemo(() => {
    return sortRows(filteredRows, sortState);
  }, [filteredRows, sortState]);

  // Dynamically calculate optimal column widths from result dataset
  const columnWidths = useMemo(() => {
    return computeColumnWidths(COLUMNS, sortedRows, {
      basePadding: 24,
      sampleLimit: 300,
      sortIndicatorWidth: 16,
    });
  }, [sortedRows]);

  const totalPages = Math.max(1, Math.ceil(sortedRows.length / PAGE_SIZE));
  const normalizedPage = Math.min(page, totalPages);

  const pageRows = useMemo(
    () => sortedRows.slice((normalizedPage - 1) * PAGE_SIZE, normalizedPage * PAGE_SIZE),
    [sortedRows, normalizedPage],
  );

  const visibleStart = sortedRows.length === 0 ? 0 : (normalizedPage - 1) * PAGE_SIZE + 1;
  const visibleEnd = Math.min(normalizedPage * PAGE_SIZE, sortedRows.length);

  const handleSort = (key: ColumnKey) => {
    setSortState((prev) => getNextSortState(prev, key));
    setPage(1);
  };

  const renderCell = (row: DataFetchRow, key: ColumnKey) => {
    switch (key) {
      case 'entry_id':
        return (
          <span className="block truncate font-mono font-medium text-foreground text-xs" title={row.entry_id}>
            {row.entry_id}
          </span>
        );
      case 'test_category':
        return row.test_category ? (
          <span className="inline-flex items-center justify-center px-1.5 py-0.5 rounded text-[11px] font-medium bg-muted/70 text-foreground border border-border/40 leading-tight">
            {row.test_category}
          </span>
        ) : (
          <span className="text-muted-foreground">-</span>
        );
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

  const activeSortCol = sortState ? COLUMNS.find((c) => c.key === sortState.key) : null;

  return (
    <div className="result-table-shell flex flex-col gap-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted-foreground">
            已显示 <strong className="font-semibold text-foreground">{visibleStart}-{visibleEnd}</strong> / 共 {sortedRows.length} 条
          </span>
          {filteredRows.length !== rows.length && (
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
              已过滤（全量 {rows.length}）
            </Badge>
          )}
          {sortState && activeSortCol && (
            <Badge
              variant="outline"
              className="text-[10px] px-1.5 py-0 cursor-pointer hover:bg-destructive/10 hover:text-destructive gap-1 transition-colors"
              onClick={() => setSortState(null)}
              title="点击恢复默认排序"
            >
              <span>{activeSortCol.label} {sortState.direction === 'asc' ? '升序 ↑' : '降序 ↓'}</span>
              <span className="text-muted-foreground">✕</span>
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-8 px-2.5 text-xs text-muted-foreground hover:text-foreground gap-1.5 shadow-none"
            onClick={() => setIsCompact((prev) => !prev)}
            title={isCompact ? '切换为等比撑满容器' : '切换为根据内容自适应紧凑列宽'}
          >
            {isCompact ? (
              <>
                <Maximize2 className="h-3.5 w-3.5" />
                <span>铺满</span>
              </>
            ) : (
              <>
                <Minimize2 className="h-3.5 w-3.5" />
                <span>紧凑</span>
              </>
            )}
          </Button>

          <div className="relative w-full sm:w-[260px]">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              placeholder="搜索条目或类别..."
              className="h-8 pl-8 text-xs"
              value={searchTerm}
              onChange={(event) => {
                setSearchTerm(event.target.value);
                setPage(1);
              }}
            />
          </div>
          {hasPendingSearch && (
            <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground shrink-0" />
          )}
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border bg-card shadow-xs">
        <Table containerClassName="max-h-[560px] overflow-auto" aria-label="数据提取结果">
          <colgroup>
            {COLUMNS.map((col) => {
              const width = columnWidths[String(col.key)];
              return (
                <col
                  key={String(col.key)}
                  style={isCompact ? { width: `${width}px`, minWidth: `${width}px` } : undefined}
                />
              );
            })}
            {isCompact && <col style={{ width: 'auto' }} />}
          </colgroup>
          <TableHeader>
            <TableRow>
              {COLUMNS.map((column) => {
                const isActive = sortState?.key === column.key;
                return (
                  <TableHead
                    key={String(column.key)}
                    style={isCompact ? { width: `${columnWidths[String(column.key)]}px`, minWidth: `${columnWidths[String(column.key)]}px` } : undefined}
                    className={cn(
                      'h-9 whitespace-nowrap text-xs cursor-pointer select-none transition-colors hover:bg-muted/80 group',
                      column.align === 'right' && 'text-right',
                      column.align === 'center' && 'text-center',
                      isActive && 'text-primary font-semibold',
                    )}
                    onClick={() => handleSort(column.key as ColumnKey)}
                    title={`点击按 ${column.label} 排序`}
                  >
                    <div
                      className={cn(
                        'inline-flex items-center gap-1.5',
                        column.align === 'right' && 'justify-end w-full',
                        column.align === 'center' && 'justify-center w-full',
                      )}
                    >
                      <span>{column.label}</span>
                      <SortIndicator
                        active={isActive}
                        direction={isActive ? sortState.direction : undefined}
                      />
                    </div>
                  </TableHead>
                );
              })}
              {isCompact && <TableHead className="p-0 border-b border-border shadow-[0_1px_0_0_var(--border)]" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {pageRows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={COLUMNS.length + (isCompact ? 1 : 0)} className="h-32 text-center text-sm text-muted-foreground">
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
                      key={String(column.key)}
                      style={isCompact ? { width: `${columnWidths[String(column.key)]}px`, minWidth: `${columnWidths[String(column.key)]}px` } : undefined}
                      className={cn(
                        'whitespace-nowrap text-xs',
                        column.align === 'right' && 'text-right font-mono tabular-nums',
                        column.align === 'center' && 'text-center',
                      )}
                    >
                      {renderCell(row, column.key as ColumnKey)}
                    </TableCell>
                  ))}
                  {isCompact && <TableCell className="p-0 border-b border-border/40" />}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
        <TablePager page={normalizedPage} totalPages={totalPages} totalRows={sortedRows.length} onPageChange={setPage} />
      </div>
    </div>
  );
}


