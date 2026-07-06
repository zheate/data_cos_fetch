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
import { Info, Search, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

const COLUMNS = [
  { key: 'device_id', label: '器件号' },
  { key: 'warehouse', label: '仓库', desc: 'COS当前存放的物理或逻辑仓位' },
  { key: 'isolation', label: '是否隔离', desc: '标识该批次是否处于隔离待检状态' },
  { key: 'item_num', label: 'ItemNum', desc: '对应的大类或批次索引' },
  { key: 'box_num', label: '盒号' },
  { key: 'owner', label: '货主' },
  { key: 'two_a_wavelength_nm', label: '2A', desc: '在 2A 典型电流下的实测波长(nm)' },
  { key: 'center_wavelength_nm', label: '中心', desc: '中心波长(nm)' },
  { key: 'cold_wavelength_nm', label: '冷波长', desc: '通常指在冷台或基准温度测试的起始波长(nm)' },
  { key: 'peak_wavelength_nm', label: '峰值', desc: '峰值波长(nm)' },
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

  const virtualizer = useVirtualizer({
    count: filteredRows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  });

  const totalCount = rows.length;
  const filteredCount = filteredRows.length;

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

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 max-w-sm">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            type="search"
            placeholder="搜索器件号、盒号或 ItemNum..."
            className="pl-9 h-9"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
        {searchTerm !== deferredSearchTerm && (
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        )}
      </div>
      <div className="overflow-hidden rounded border bg-card shadow-sm">
      <div ref={scrollRef} className="overflow-auto" style={{ maxHeight: MAX_HEIGHT }}>
        <Table aria-label={label ?? 'COS data'}>
          <TableHeader className="sticky top-0 z-10 bg-muted/80 backdrop-blur">
            <TableRow>
              <TooltipProvider>
                {COLUMNS.map((column) => (
                  <TableHead key={column.key} className={column.key.includes('nm') ? 'text-right' : undefined}>
                    {column.desc ? (
                      <Tooltip>
                        <TooltipTrigger className="inline-flex cursor-help items-center gap-1.5 focus:outline-none">
                          {column.label}
                          <Info className="h-3.5 w-3.5 text-muted-foreground/60" />
                        </TooltipTrigger>
                        <TooltipContent side="top">
                          <p className="max-w-[200px] leading-relaxed">{column.desc}</p>
                        </TooltipContent>
                      </Tooltip>
                    ) : (
                      column.label
                    )}
                  </TableHead>
                ))}
              </TooltipProvider>
            </TableRow>
          </TableHeader>
          <TableBody>
            {/* Top spacer to offset scrolled-past rows */}
            {virtualizer.getVirtualItems().length > 0 && (
              <tr style={{ height: virtualizer.getVirtualItems()[0].start }} />
            )}
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const row = filteredRows[virtualRow.index];
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
                      key={column.key}
                      className={column.key.includes('nm') ? 'text-right tabular-nums' : undefined}
                    >
                      {renderCell(row, column.key)}
                    </TableCell>
                  ))}
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
              />
            )}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-between border-t px-4 py-2 text-sm text-muted-foreground">
        <span>
          {searchTerm.trim() ? `搜索到 ${filteredCount} 条（共 ${totalCount} 条）` : `共 ${totalCount} 条`}
        </span>
      </div>
    </div>
    </div>
  );
}
