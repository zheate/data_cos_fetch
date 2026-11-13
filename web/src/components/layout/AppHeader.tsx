import { Activity, Database, Filter, FolderOpen, Moon, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';
import { useAppStore } from '../../stores/app-store';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

const NAV_ITEMS = [
  { id: 'data_fetch', label: '数据提取', icon: Database },
  { id: 'cos_filter', label: 'COS 筛选', icon: Filter },
] as const;

export function AppHeader() {
  const { mainTab, setMainTab, logPath, backendReady, backendError, setBackendError } = useAppStore();
  const { theme, setTheme } = useTheme();

  const openLogsFolder = async () => {
    const error = await window.desktopRuntime?.openLogsFolder?.();
    if (error) setBackendError(`${backendError}\n打开日志目录失败：${error}`);
  };

  return (
    <header className="sticky top-0 z-30 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/85">
      <div className="flex h-12 w-full items-center gap-3 px-4 sm:gap-10 sm:px-6">
        <button
          type="button"
          onClick={() => setMainTab('data_fetch')}
          className="flex shrink-0 items-center gap-2.5 rounded-sm text-sm font-semibold text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
          aria-label="返回数据提取"
        >
          <Activity className="size-5 shrink-0 text-muted-foreground" />
          <span className="whitespace-nowrap sm:hidden">COS</span>
          <span className="hidden whitespace-nowrap sm:inline">Data COS Suite</span>
        </button>

        <nav className="flex h-full shrink-0 items-stretch gap-1" aria-label="主要导航">
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon;
            const active = mainTab === item.id;
            return (
              <button
                key={item.id}
                type="button"
                aria-current={active ? 'page' : undefined}
                onClick={() => setMainTab(item.id)}
                className={cn(
                  'my-auto flex h-8 shrink-0 items-center gap-2 whitespace-nowrap rounded-md px-2.5 text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 sm:px-3',
                  active
                    ? 'bg-secondary font-medium text-secondary-foreground'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                )}
              >
                <Icon className="size-3.5" />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        <div className="ml-auto flex items-center gap-1.5">
          <span className="mr-2 hidden items-center gap-1.5 text-[11px] text-muted-foreground md:flex" title={backendError || undefined}>
            <span className={cn('size-1.5 rounded-full', backendError ? 'bg-destructive' : backendReady ? 'bg-success' : 'animate-pulse bg-muted-foreground')} />
            {backendError ? '服务异常' : backendReady ? '核心服务在线' : '连接中'}
          </span>
          {window.desktopRuntime?.openLogsFolder && logPath && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={openLogsFolder}
              className="text-muted-foreground hover:text-foreground"
              title={`日志路径：${logPath}`}
            >
              <FolderOpen />
              <span className="hidden sm:inline">运行日志</span>
            </Button>
          )}
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            aria-label="切换主题"
            title="切换主题"
          >
            {theme === 'dark' ? <Sun className="text-warning" /> : <Moon />}
          </Button>
        </div>
      </div>
    </header>
  );
}
