import { Database, Filter, Moon, Sun, Settings, ShieldAlert } from 'lucide-react';
import { useTheme } from 'next-themes';
import { useAppStore } from '../stores/app-store';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';

export function AppNavbar() {
  const { mainTab, setMainTab } = useAppStore();
  const { theme, setTheme } = useTheme();

  return (
    <header className="app-navbar sticky top-0 z-40 border-b bg-background/95 supports-[backdrop-filter]:bg-background/60 backdrop-blur select-none">
      <div className="mx-auto flex w-full max-w-[1600px] items-center justify-between gap-4 px-4 py-2 sm:px-6 lg:px-8">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex size-8 shrink-0 items-center justify-center rounded border bg-primary text-primary-foreground shadow-sm">
            <Database className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-bold tracking-tight text-foreground">
              Data CoS Suite
            </p>
            <p className="truncate text-[10px] font-medium tracking-wider uppercase text-muted-foreground">
              数据工作台
            </p>
          </div>
        </div>

        <Tabs value={mainTab} onValueChange={(value) => value === 'data_fetch' || value === 'cos_filter' ? setMainTab(value) : null}>
          <TabsList className="h-8 p-0.5 bg-muted/60 rounded">
            <TabsTrigger value="data_fetch" className="rounded-sm px-3.5 py-1 text-xs">
              <Database className="h-3.5 w-3.5 mr-1" />
              数据提取
            </TabsTrigger>
            <TabsTrigger value="cos_filter" className="rounded-sm px-3.5 py-1 text-xs">
              <Filter className="h-3.5 w-3.5 mr-1" />
              COS 筛选
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="rounded"
            title="系统审计 (Audit Log)"
          >
            <ShieldAlert className="h-3.5 w-3.5 text-muted-foreground" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="rounded"
            title="配置中心 (Configuration)"
          >
            <Settings className="h-3.5 w-3.5 text-muted-foreground" />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            aria-label="Toggle theme"
            className="rounded size-7"
          >
            {theme === 'dark' ? <Sun className="h-3.5 w-3.5 text-warning" /> : <Moon className="h-3.5 w-3.5 text-primary" />}
          </Button>
        </div>
      </div>
    </header>
  );
}
