import { useRef, useMemo, useState, useDeferredValue } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import type { CosRow } from '../helpers/types';
import { toFixed } from '../helpers/utils';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  Info,
  Search,
  Loader2,
  Maximize2,
  Minimize2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  computeColumnWidths,
  getNextSortState,
  sortRows,
  type ColumnWidthSpec,
  type SortState,
} from '../helpers/tableLayout';

interface CosColumnSpec extends ColumnWidthSpec<CosRow> {
  desc?: string;
}

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

const COLUMNS: CosColumnSpec[] = [
  { key: 'device_id', label: '器件号', isMono: true, minWidth: 105, maxWidth: 220, getText: (r) => r.device_id },
  { key: 'warehouse', label: '仓库', desc: 'COS当前存放的物理或逻辑仓位', minWidth: 70, extraPadding: 18, getText: (r) => r.warehouse ?? '' },
  { key: 'isolation', label: '是否隔离', desc: '标识该批次是否处于隔离待检状态', minWidth: 72, extraPadding: 18, getText: (r) => r.isolation ?? '' },
  { key: 'item_num', label: 'ItemNum', desc: '对应的大类或批次索引', minWidth: 75, extraPadding: 18, getText: (r) => r.item_num ?? '' },
  { key: 'box_num', label: '盒号', minWidth: 70, getText: (r) => r.box_num ?? '' },
  { key: 'owner', label: '货主', minWidth: 70, getText: (r) => r.owner ?? '' },
  { key: 'two_a_wavelength_nm', label: '2A', desc: '在 2A 典型电流下的实测波长(nm)', align: 'right', isMono: true, minWidth: 68, extraPadding: 18, getText: (r) => toFixed(r.two_a_wavelength_nm, 3) },
  { key: 'center_wavelength_nm', label: '中心', desc: '中心波长(nm)', align: 'right', isMono: true, minWidth: 68, extraPadding: 18, getText: (r) => toFixed(r.center_wavelength_nm, 3) },
  { key: 'cold_wavelength_nm', label: '冷波长', desc: '通常指在冷台或基准温度测试的起始波长(nm)', align: 'right', isMono: true, minWidth: 72, extraPadding: 18, getText: (r) => toFixed(r.cold_wavelength_nm, 3) },
  { key: 'peak_wavelength_nm', label: '峰值', desc: '峰值波长(nm)', align: 'right', isMono: true, minWidth: 68, extraPadding: 18, getText: (r) => toFixed(r.peak_wavelength_nm, 3) },
];

const ROW_HEIGHT = 36;
const MAX_HEIGHT = 460;

export function CosTable({
  rows,
  label,
  activeRow,
  onRowClick,
}: {
  rows: CosRow[];
  label?: string;
  activeRow?: CosRow | null;
  onRowClick?: (row: CosRow) => void;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [isCompact, setIsCompact] = useState(true);
  const [sortState, setSortState] = useState<SortState<string> | null>(null);
  const deferredSearchTerm = useDeferredValue(searchTerm);

  const filteredRows = useMemo(() => {
    if (!deferredSearchTerm.trim()) return rows;
    const lower = deferredSearchTerm.toLowerCase();
    return rows.filter((row) =>
      row.device_id.toLowerCase().includes(lower) ||
      (row.box_num && row.box_num.toLowerCase().includes(lower)) ||
      (row.item_num && row.item_num.toLowerCase().includes(lower))
    );
  }, [rows, deferredSearchTerm]);

  // Sort rows based on active header sort
  const sortedRows = useMemo(() => {
    return sortRows(filteredRows, sortState as SortState<keyof CosRow> | null);
  }, [filteredRows, sortState]);

  // Dynamically compute column widths based on results
  const columnWidths = useMemo(() => {
    return computeColumnWidths(COLUMNS, sortedRows, {
      basePadding: 24,
      sampleLimit: 300,
      sortIndicatorWidth: 16,
    });
  }, [sortedRows]);

  const virtualizer = useVirtualizer({
    count: sortedRows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  });

  const totalCount = rows.length;
  const filteredCount = sortedRows.length;

  const handleSort = (key: string) => {
    setSortState((prev) => getNextSortState(prev, key));
  };

  const renderCell = useMemo(
    () => (row: CosRow, key: string) => {
      switch (key) {
        case 'device_id':
          return row.device_id;
        case 'warehouse':
          return row.warehouse ?? '-';
        case 'isolation':
          return row.isolation ?? '-';
        case 'item_num':
          return row.item_num ?? '-';
        case 'box_num':
          return row.box_num ?? '-';
        case 'owner':
          return row.owner ?? '-';
        case 'two_a_wavelength_nm':
          return toFixed(row.two_a_wavelength_nm, 3);
        case 'center_wavelength_nm':
          return toFixed(row.center_wavelength_nm, 3);
        case 'cold_wavelength_nm':
          return toFixed(row.cold_wavelength_nm, 3);
        case 'peak_wavelength_nm':
          return toFixed(row.peak_wavelength_nm, 3);
        default:
          return '-';
      }
    },
    [],
  );

  if (rows.length === 0) {
    return <p className="py-6 text-center text-sm text-muted-foreground">暂无数据</p>;
  }

  const activeSortCol = sortState ? COLUMNS.find((c) => c.key === sortState.key) : null;

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted-foreground">
            显示 <strong className="font-semibold text-foreground">{filteredCount}</strong> 项
            {filteredCount !== totalCount && `（筛选自 ${totalCount} 项）`}
          </span>
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

          <div className="relative w-full sm:w-[240px]">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              placeholder="搜索器件号、盒号..."
              className="h-8 pl-8 text-xs"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
          {searchTerm !== deferredSearchTerm && (
            <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground shrink-0" />
          )}
        </div>
      </div>
      <div className="overflow-hidden rounded-lg border bg-card shadow-xs">
        <Table
          containerRef={scrollRef}
          containerStyle={{ maxHeight: MAX_HEIGHT }}
          containerClassName="overflow-auto"
          aria-label={label ?? 'COS data'}
        >
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
              <TooltipProvider>
                {COLUMNS.map((column) => {
                  const isActive = sortState?.key === column.key;
                  return (
                    <TableHead
                      key={String(column.key)}
                      style={isCompact ? { width: `${columnWidths[String(column.key)]}px`, minWidth: `${columnWidths[String(column.key)]}px` } : undefined}
                      className={cn(
                        'h-9 whitespace-nowrap text-xs cursor-pointer select-none transition-colors hover:bg-muted/80 group',
                        (column.align === 'right' || column.key.includes('nm')) && 'text-right',
                        isActive && 'text-primary font-semibold',
                      )}
                      onClick={() => handleSort(String(column.key))}
                      title={`点击按 ${column.label} 排序`}
                    >
                      <div
                        className={cn(
                          'inline-flex items-center gap-1.5',
                          (column.align === 'right' || column.key.includes('nm')) && 'justify-end w-full',
                        )}
                      >
                        {column.desc ? (
                          <Tooltip>
                            <TooltipTrigger
                              className="inline-flex cursor-help items-center gap-1 focus:outline-none"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <span>{column.label}</span>
                              <Info className="h-3 w-3 text-muted-foreground/60" />
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <p className="max-w-[200px] leading-relaxed">{column.desc}</p>
                            </TooltipContent>
                          </Tooltip>
                        ) : (
                          <span>{column.label}</span>
                        )}
                        <SortIndicator
                          active={isActive}
                          direction={isActive ? sortState.direction : undefined}
                        />
                      </div>
                    </TableHead>
                  );
                })}
              </TooltipProvider>
              {isCompact && <TableHead className="p-0 border-b border-border shadow-[0_1px_0_0_var(--border)]" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {/* Top spacer to offset scrolled-past rows */}
            {virtualizer.getVirtualItems().length > 0 && (
              <tr style={{ height: virtualizer.getVirtualItems()[0].start }}>
                <td colSpan={COLUMNS.length + (isCompact ? 1 : 0)} className="p-0 border-0" />
              </tr>
            )}
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const row = sortedRows[virtualRow.index];
              const isSelected = activeRow?.device_id === row.device_id;
              return (
                <TableRow
                  key={`${row.device_id}-${virtualRow.index}`}
                  className={cn(
                    'cursor-pointer select-none transition-colors hover:bg-muted/50',
                    isSelected && 'bg-muted hover:bg-muted/80',
                  )}
                  onClick={() => onRowClick?.(row)}
                >
                  {COLUMNS.map((column) => (
                    <TableCell
                      key={String(column.key)}
                      style={isCompact ? { width: `${columnWidths[String(column.key)]}px`, minWidth: `${columnWidths[String(column.key)]}px` } : undefined}
                      className={cn(
                        'whitespace-nowrap text-xs',
                        (column.align === 'right' || column.key.includes('nm')) && 'text-right tabular-nums font-mono',
                        column.isMono && 'font-mono',
                      )}
                    >
                      {renderCell(row, String(column.key))}
                    </TableCell>
                  ))}
                  {isCompact && <TableCell className="p-0 border-b border-border/40" />}
                </TableRow>
              );
            })}
            {/* Bottom spacer for remaining rows */}
            {virtualizer.getVirtualItems().length > 0 && (
              <tr
                style={{
                  height:
                    virtualizer.getTotalSize() -
                    (virtualizer.getVirtualItems().at(-1)?.end ?? 0),
                }}
              >
                <td colSpan={COLUMNS.length + (isCompact ? 1 : 0)} className="p-0 border-0" />
              </tr>
            )}
          </TableBody>
        </Table>
        <div className="flex items-center justify-between border-t px-4 py-2 text-sm text-muted-foreground">
          <span>
            {searchTerm.trim() ? `搜索到 ${filteredCount} 条（共 ${totalCount} 条）` : `共 ${totalCount} 条`}
          </span>
        </div>
      </div>
    </div>
  );
}

