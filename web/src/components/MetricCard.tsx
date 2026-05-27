import type { ReactNode } from 'react';
import { Card, CardContent, CardDescription } from '@/components/ui/card';
import { cn } from '@/lib/utils';

type MetricCardProps = {
  label: string;
  value: string;
  color?: 'default' | 'primary' | 'success' | 'warning' | 'danger';
  icon?: ReactNode;
};

export function MetricCard({ label, value, color = 'default', icon }: MetricCardProps) {
  const styles: Record<string, { accent: string; value: string; icon: string; bg: string }> = {
    default: {
      accent: 'bg-slate-100/80 dark:bg-slate-800/50',
      value: 'text-foreground',
      icon: 'text-slate-600 dark:text-slate-400',
      bg: 'hover:bg-slate-50/50 dark:hover:bg-slate-900/50',
    },
    primary: {
      accent: 'bg-sky-100/80 dark:bg-sky-900/30',
      value: 'text-sky-700 dark:text-sky-300',
      icon: 'text-sky-600 dark:text-sky-400',
      bg: 'hover:bg-sky-50/50 dark:hover:bg-sky-900/10',
    },
    success: {
      accent: 'bg-emerald-100/80 dark:bg-emerald-900/30',
      value: 'text-emerald-700 dark:text-emerald-300',
      icon: 'text-emerald-600 dark:text-emerald-400',
      bg: 'hover:bg-emerald-50/50 dark:hover:bg-emerald-900/10',
    },
    warning: {
      accent: 'bg-amber-100/80 dark:bg-amber-900/30',
      value: 'text-amber-700 dark:text-amber-300',
      icon: 'text-amber-600 dark:text-amber-400',
      bg: 'hover:bg-amber-50/50 dark:hover:bg-amber-900/10',
    },
    danger: {
      accent: 'bg-rose-100/80 dark:bg-rose-900/30',
      value: 'text-rose-700 dark:text-rose-300',
      icon: 'text-rose-600 dark:text-rose-400',
      bg: 'hover:bg-rose-50/50 dark:hover:bg-rose-900/10',
    },
  };

  const scheme = styles[color] ?? styles.default;

  return (
    <Card className={cn("relative overflow-hidden border-border/50 py-0 shadow-sm transition-all duration-300 hover:shadow-md hover:-translate-y-0.5", scheme.bg)}>
      <div className="absolute inset-x-0 -top-px h-px w-full bg-gradient-to-r from-transparent via-primary/10 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
      <CardContent className="flex items-start justify-between gap-4 p-5">
        <div className="flex flex-col gap-2.5">
          <CardDescription className="text-xs font-semibold uppercase tracking-[0.15em] text-muted-foreground">
            {label}
          </CardDescription>
          <strong className={cn('text-3xl font-bold tracking-tight tabular-nums transition-colors', scheme.value)}>
            {value}
          </strong>
        </div>
        {icon && (
          <div className={cn('flex size-11 shrink-0 items-center justify-center rounded-xl border border-background/20 transition-transform duration-300 group-hover:scale-110 shadow-sm', scheme.accent, scheme.icon)}>
            {icon}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
