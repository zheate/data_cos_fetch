import { ChevronRight, FolderOpen, Moon, Sparkles, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';
import { toast } from 'sonner';
import { useAppStore } from '../../stores/app-store';
import { Button } from '@/components/ui/button';
import { injectAllMockData, clearAllMockData } from '../../helpers/mockData';

export function AppHeader() {
  const { mainTab, logPath, backendError, setBackendError } = useAppStore();
  const { theme, setTheme } = useTheme();

  const openLogsFolder = async () => {
    const error = await window.desktopRuntime?.openLogsFolder?.();
    if (error) setBackendError(`${backendError}\n打开日志目录失败：${error}`);
  };

  const title = mainTab === 'data_fetch' ? '数据提取' : 'COS 筛选';

  return (
    <header className="sticky top-0 z-20 flex h-12 shrink-0 items-center justify-between border-b bg-background/95 px-4 sm:px-6 backdrop-blur supports-[backdrop-filter]:bg-background/60 select-none">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-2 text-xs">
        <span className="text-muted-foreground">工作台</span>
        <ChevronRight className="size-3.5 text-muted-foreground/60" />
        <span className="font-semibold text-foreground">{title}</span>
      </div>

      {/* Right controls */}
      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="xs"
          onClick={() => {
            injectAllMockData();
            toast.success('已注入全套虚拟数据，可立即查看数据明细与图表！');
          }}
          className="h-7 text-xs gap-1.5 px-2.5 font-medium border-primary/25 hover:border-primary/50 text-foreground shadow-xs"
          title="注入高仿真的实测记录与成组结果"
        >
          <Sparkles className="size-3 text-amber-500" />
          <span>注入演示数据</span>
        </Button>

        <Button
          type="button"
          variant="ghost"
          size="xs"
          onClick={() => {
            clearAllMockData();
            toast.info('已重置清空工作台数据');
          }}
          className="h-7 text-xs px-2 text-muted-foreground hover:text-destructive"
          title="清空所有测试与分组数据"
        >
          清空
        </Button>

        {window.desktopRuntime?.openLogsFolder && logPath && (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            onClick={openLogsFolder}
            className="text-xs text-muted-foreground hover:text-foreground h-7 gap-1.5 px-2"
            title={`日志路径：${logPath}`}
          >
            <FolderOpen className="size-3.5" />
            <span className="hidden sm:inline">运行日志</span>
          </Button>
        )}

        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
          aria-label="切换主题"
          className="size-7 rounded"
        >
          {theme === 'dark' ? <Sun className="size-3.5 text-warning" /> : <Moon className="size-3.5 text-muted-foreground" />}
        </Button>
      </div>
    </header>
  );
}
