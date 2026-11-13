import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';

type TablePagerProps = {
  page: number;
  totalPages: number;
  totalRows: number;
  pageSize?: number;
  onPageChange: (page: number) => void;
};

export function TablePager({ page, totalPages, totalRows, pageSize, onPageChange }: TablePagerProps) {
  if (totalPages <= 1 && !pageSize) {
    return null;
  }

  const start = totalRows === 0 ? 0 : (page - 1) * (pageSize ?? totalRows) + 1;
  const end = Math.min(page * (pageSize ?? totalRows), totalRows);

  return (
    <div className="flex min-h-11 items-center justify-between gap-3 border-t px-4 py-1.5">
      <p className="text-[11px] text-muted-foreground">
        {pageSize ? `第 ${start}–${end} 条，共 ${totalRows} 条` : `第 ${page} / ${totalPages} 页，共 ${totalRows} 条`}
      </p>
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
          aria-label="上一页"
          title="上一页"
          className="text-muted-foreground"
        >
          <ChevronLeft />
        </Button>
        <span className="min-w-12 text-center text-[11px] tabular-nums text-muted-foreground">{page} / {totalPages}</span>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
          aria-label="下一页"
          title="下一页"
          className="text-muted-foreground"
        >
          <ChevronRight />
        </Button>
      </div>
    </div>
  );
}
