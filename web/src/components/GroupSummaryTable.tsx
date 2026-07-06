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

const COLUMNS = [
  { key: 'groupId', label: '分组编号' },
  { key: 'count', label: '组内项数' },
  { key: 'avg', label: '平均波长 (nm)' },
  { key: 'min', label: '最小值 (nm)' },
  { key: 'max', label: '最大值 (nm)' },
  { key: 'diff', label: '极差值 (nm)' },
  { key: 'cv', label: '离散度 CV (%)' },
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
  if (rows.length === 0) {
    return <p className="rounded-lg border bg-muted/20 py-8 text-center text-sm text-muted-foreground">暂无分组数据</p>;
  }

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
    <div className="overflow-hidden rounded-lg border bg-card shadow-sm">
      <div className="max-h-[420px] overflow-auto">
        <Table aria-label="Group summary">
          <TableHeader className="sticky top-0 z-10 bg-muted/80 backdrop-blur">
            <TableRow>
              {COLUMNS.map((column) => (
                <TableHead key={column.key} className={column.key !== 'groupId' && column.key !== 'count' ? 'text-right' : undefined}>
                  {column.label}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row, index) => {
              const isSelected = (selectedGroupIndex ?? 0) === index;
              return (
                <TableRow
                  key={row.groupId}
                  className={cn('cursor-pointer', isSelected && 'bg-muted hover:bg-muted/80') }
                  onClick={() => onSelectionChange?.(index)}
                >
                  {COLUMNS.map((column) => (
                    <TableCell
                      key={column.key}
                      className={cn(
                        column.key !== 'groupId' && column.key !== 'count' ? 'text-right tabular-nums' : undefined,
                        isSelected && 'font-medium',
                      )}
                    >
                      {renderCell(row, column.key)}
                    </TableCell>
                  ))}
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
