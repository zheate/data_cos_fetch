import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

type EmptyStateProps = {
  icon: ReactNode;
  title: string;
  description?: ReactNode;
  notes?: string[];
  className?: string;
};

export function EmptyState({ icon, title, description, notes = [], className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center text-center border border-dashed rounded-lg bg-muted/5 px-6 py-12', className)}>
      <div className="flex size-12 items-center justify-center rounded-lg border bg-background text-muted-foreground/80 shadow-sm">
        <div className="size-5 [&_svg]:size-5">
          {icon}
        </div>
      </div>
      <h3 className="text-sm font-semibold mt-4 text-foreground">{title}</h3>
      {description && (
        <p className="max-w-md text-xs text-muted-foreground mt-1.5 leading-relaxed">{description}</p>
      )}
      {notes.length > 0 && (
        <div className="mx-auto mt-6 flex max-w-md flex-col gap-2.5 rounded-lg border bg-muted/20 p-4 text-left text-xs text-muted-foreground">
          {notes.map((note) => (
            <p key={note} className="flex items-start gap-2 leading-relaxed">
              <span className="mt-1.5 flex size-1.5 shrink-0 rounded-full bg-muted-foreground/45" />
              <span>{note}</span>
            </p>
          ))}
        </div>
      )}
    </div>
  );
}
