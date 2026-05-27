import { Database, Filter, Moon, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';
import { useAppStore } from '../stores/app-store';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';

export function AppNavbar() {
  const { mainTab, setMainTab } = useAppStore();
  const { theme, setTheme } = useTheme();

  return (
    <header className="sticky top-0 z-40 border-b border-border/40 bg-background/70 backdrop-blur-xl supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto flex w-full max-w-[1600px] items-center justify-between gap-4 px-4 py-3 sm:px-6 lg:px-8">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-primary/10 border border-primary/20 text-primary shadow-sm transition-transform hover:scale-105">
            <Filter />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold tracking-tight">Data CoS Suite</p>
            <p className="truncate text-xs text-muted-foreground">数据工作台</p>
          </div>
        </div>

        <Tabs value={mainTab} onValueChange={(value) => value === 'data_fetch' || value === 'cos_filter' ? setMainTab(value) : null}>
          <TabsList variant="default" className="bg-muted/70">
            <TabsTrigger value="data_fetch">
              <Database data-icon="inline-start" />
              数据提取
            </TabsTrigger>
            <TabsTrigger value="cos_filter">
              <Filter data-icon="inline-start" />
              COS 筛选
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            aria-label="Toggle theme"
          >
            {theme === 'dark' ? <Sun /> : <Moon />}
          </Button>
        </div>
      </div>
    </header>
  );
}
