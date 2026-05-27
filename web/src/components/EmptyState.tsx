import type { ReactNode } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
    <Card className={cn('group relative overflow-hidden border-dashed border-border/60 bg-muted/10 shadow-sm transition-colors hover:bg-muted/20', className)}>
      <div className="absolute inset-0 bg-gradient-to-b from-transparent to-muted/10 pointer-events-none" />
      <CardHeader className="items-center text-center pb-6 pt-10">
        <div className="relative mb-4 flex size-16 items-center justify-center rounded-2xl border border-background/20 bg-card text-muted-foreground shadow-sm transition-transform duration-500 group-hover:scale-110 group-hover:text-primary">
          <div className="absolute inset-0 -z-10 rounded-full bg-primary/10 blur-[20px] transition-opacity duration-500 group-hover:opacity-100 opacity-0" />
          {icon}
        </div>
        <CardTitle className="text-xl tracking-tight">{title}</CardTitle>
        <CardDescription className="max-w-xl text-[15px] leading-relaxed">{description}</CardDescription>
      </CardHeader>
      {notes.length > 0 && (
        <CardContent className="pb-10">
          <div className="mx-auto flex max-w-xl flex-col gap-2.5 rounded-xl border border-border/50 bg-background/50 p-5 text-sm text-muted-foreground backdrop-blur-sm transition-colors group-hover:border-border/80">
            {notes.map((note) => (
              <p key={note} className="flex items-start gap-2">
                <span className="mt-1 flex size-1.5 shrink-0 rounded-full bg-primary/50" />
                <span>{note}</span>
              </p>
            ))}
          </div>
        </CardContent>
      )}
    </Card>
  );
}
