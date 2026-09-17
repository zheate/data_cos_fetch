import { useMemo, useState } from 'react';
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react';
import type { GroupSummaryRow } from '../helpers/types';
import { toFixed } from '../helpers/utils';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn } from '@/lib/utils';
import {
  computeColumnWidths,
  getNextSortState,
  sortRows,
  type ColumnWidthSpec,
  type SortState,
} from '../helpers/tableLayout';

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

const COLUMNS: ColumnWidthSpec<GroupSummaryRow>[] = [
  { key: 'groupId', label: '分组编号', minWidth: 85, getText: (r) => r.groupId },
  { key: 'count', label: '组内项数', align: 'right', minWidth: 80, getText: (r) => `${r.count} pcs` },
  { key: 'avg', label: '平均波长 (nm)', align: 'right', isMono: true, minWidth: 100, getText: (r) => toFixed(r.avg, 3) },
  { key: 'min', label: '最小值 (nm)', align: 'right', isMono: true, minWidth: 90, getText: (r) => toFixed(r.min, 3) },
  { key: 'max', label: '最大值 (nm)', align: 'right', isMono: true, minWidth: 90, getText: (r) => toFixed(r.max, 3) },
  { key: 'diff', label: '极差值 (nm)', align: 'right', isMono: true, minWidth: 90, getText: (r) => toFixed(r.diff, 3) },
  { key: 'cv', label: '离散度 CV (%)', align: 'right', isMono: true, minWidth: 96, getText: (r) => toFixed(r.cv, 4) },
];

export function GroupSummaryTable({
  rows,
  selectedGroupIndex,
  onSelectionChange,
}: {
  rows: GroupSummaryRow[];
  selectedGroupIndex?: number;
  onSelectionChange?: (index: number) => void;
}) {
  const [sortState, setSortState] = useState<SortState<keyof GroupSummaryRow> | null>(null);

  const sortedRows = useMemo(() => {
    return sortRows(rows, sortState);
  }, [rows, sortState]);

  const columnWidths = useMemo(() => {
    return computeColumnWidths(COLUMNS, sortedRows, {
      basePadding: 24,
      sortIndicatorWidth: 16,
    });
  }, [sortedRows]);

  if (rows.length === 0) {
    return <p className="rounded-lg border bg-muted/20 py-8 text-center text-sm text-muted-foreground">暂无分组数据</p>;
  }

  const handleSort = (key: keyof GroupSummaryRow) => {
    setSortState((prev) => getNextSortState(prev, key));
  };

  const renderCell = (row: GroupSummaryRow, key: string) => {
    switch (key) {
      case 'groupId':
        return <span className="font-medium text-primary tracking-tight px-1">{row.groupId}</span>;
      case 'count':
        return <span className="font-medium text-foreground">{row.count} <span className="text-muted-foreground font-normal">pcs</span></span>;
      case 'avg':
        return toFixed(row.avg, 3);
      case 'min':
        return toFixed(row.min, 3);
      case 'max':
        return toFixed(row.max, 3);
      case 'diff':
        return (
          <span className={row.diff !== null && row.diff > 1.0 ? "text-warning font-medium" : "text-success"}>
            {toFixed(row.diff, 3)}
          </span>
        );
      case 'cv':
        return toFixed(row.cv, 4);
      default:
        return '-';
    }
  };

  return (
    <div className="overflow-hidden rounded-lg border bg-card shadow-xs">
      <Table containerClassName="max-h-[420px] overflow-auto" aria-label="Group summary">
        <colgroup>
          {COLUMNS.map((col) => {
            const width = columnWidths[String(col.key)];
            return (
              <col
                key={String(col.key)}
                style={{ width: `${width}px`, minWidth: `${width}px` }}
              />
            );
          })}
          <col style={{ width: 'auto' }} />
        </colgroup>
        <TableHeader>
          <TableRow>
            {COLUMNS.map((column) => {
              const isActive = sortState?.key === column.key;
              return (
                <TableHead
                  key={String(column.key)}
                  style={{ width: `${columnWidths[String(column.key)]}px`, minWidth: `${columnWidths[String(column.key)]}px` }}
                  className={cn(
                    'h-9 whitespace-nowrap text-xs cursor-pointer select-none transition-colors hover:bg-muted/80 group',
                    column.align === 'right' && 'text-right',
                    isActive && 'text-primary font-semibold',
                  )}
                  onClick={() => handleSort(column.key)}
                  title={`点击按 ${column.label} 排序`}
                >
                  <div
                    className={cn(
                      'inline-flex items-center gap-1.5',
                      column.align === 'right' && 'justify-end w-full',
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
            <TableHead className="p-0 border-b border-border shadow-[0_1px_0_0_var(--border)]" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedRows.map((row, index) => {
            const isSelected = (selectedGroupIndex ?? 0) === index;
            return (
              <TableRow
                key={row.groupId}
                className={cn('cursor-pointer', isSelected && 'bg-muted hover:bg-muted/80')}
                onClick={() => onSelectionChange?.(index)}
              >
                {COLUMNS.map((column) => (
                  <TableCell
                    key={String(column.key)}
                    style={{ width: `${columnWidths[String(column.key)]}px`, minWidth: `${columnWidths[String(column.key)]}px` }}
                    className={cn(
                      'whitespace-nowrap text-xs',
                      column.align === 'right' && 'text-right tabular-nums font-mono',
                      isSelected && 'font-medium',
                    )}
                  >
                    {renderCell(row, String(column.key))}
                  </TableCell>
                ))}
                <TableCell className="p-0 border-b border-border/40" />
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

