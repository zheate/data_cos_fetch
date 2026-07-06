import type { ReactNode } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';

type MetricCardProps = {
  label: string;
  value: string;
  color?: 'default' | 'primary' | 'success' | 'warning' | 'danger';
  icon?: ReactNode;
};

export function MetricCard({ label, value, color = 'default', icon }: MetricCardProps) {
  const textColorMap: Record<string, string> = {
    default: 'text-foreground',
    primary: 'text-foreground',
    success: 'text-success',
    warning: 'text-warning',
    danger: 'text-destructive',
  };

  const iconColorMap: Record<string, string> = {
    default: 'text-muted-foreground',
    primary: 'text-muted-foreground',
    success: 'text-success',
    warning: 'text-warning',
    danger: 'text-destructive',
  };

  return (
    <Card className="rounded border bg-card text-card-foreground shadow-sm">
      <CardContent className="p-4">
        <div className="flex items-center justify-between space-y-0 pb-1">
          <span className="text-[11px] font-semibold text-muted-foreground tracking-wider uppercase">{label}</span>
          {icon && (
            <div className={cn("size-3.5 shrink-0 [&_svg]:size-3.5", iconColorMap[color] ?? iconColorMap.default)}>
              {icon}
            </div>
          )}
        </div>
        <div className={cn("text-xl font-bold tracking-tight", textColorMap[color] ?? textColorMap.default)}>
          {value}
        </div>
      </CardContent>
    </Card>
  );
}
