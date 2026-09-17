import { Database, Filter, ChevronLeft, ChevronRight } from 'lucide-react';
import { useAppStore } from '../../stores/app-store';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

type NavItem = {
  id: 'data_fetch' | 'cos_filter';
  label: string;
  description: string;
  icon: typeof Database;
};

const NAV_ITEMS: NavItem[] = [
  {
    id: 'data_fetch',
    label: '数据提取',
    description: '原始数据扫描与解析',
    icon: Database,
  },
  {
    id: 'cos_filter',
    label: 'COS 筛选',
    description: '波长聚类与自动成组',
    icon: Filter,
  },
];

export function AppSidebar({
  collapsed,
  onToggleCollapsed,
}: {
  collapsed: boolean;
  onToggleCollapsed: () => void;
}) {
  const { mainTab, setMainTab, backendReady, backendError } = useAppStore();

  return (
    <TooltipProvider delayDuration={150}>
      <aside
        className={cn(
          'relative flex flex-col border-r bg-sidebar text-sidebar-foreground transition-[width] duration-200 select-none shrink-0 z-30',
          collapsed ? 'w-14' : 'w-56',
        )}
      >
        {/* Header / Brand */}
        <div className="flex h-12 items-center border-b px-3 justify-between">
          <div className="flex items-center gap-2.5 overflow-hidden">
            <span className="flex size-7 shrink-0 items-center justify-center rounded bg-primary text-xs font-bold text-primary-foreground">
              COS
            </span>
            {!collapsed && (
              <div className="flex flex-col min-w-0">
                <span className="truncate text-xs font-semibold tracking-tight text-foreground">
                  Data CoS Suite
                </span>
                <span className="truncate text-[10px] text-muted-foreground">
                  工程分析套件
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Navigation items */}
        <nav className="flex-1 space-y-1 p-2">
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon;
            const isActive = mainTab === item.id;

            const buttonContent = (
              <button
                type="button"
                onClick={() => setMainTab(item.id)}
                className={cn(
                  'group flex w-full items-center rounded-md px-2.5 py-2 text-left text-xs transition-colors',
                  collapsed ? 'justify-center px-0' : 'gap-3',
                  isActive
                    ? 'bg-sidebar-accent text-sidebar-accent-foreground font-semibold shadow-xs'
                    : 'text-muted-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-foreground',
                )}
              >
                <Icon
                  className={cn(
                    'size-4 shrink-0 transition-colors',
                    isActive ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground',
                  )}
                />
                {!collapsed && (
                  <div className="flex min-w-0 flex-col">
                    <span className="truncate">{item.label}</span>
                    <span className="truncate text-[10px] font-normal text-muted-foreground/80">
                      {item.description}
                    </span>
                  </div>
                )}
              </button>
            );

            if (collapsed) {
              return (
                <Tooltip key={item.id}>
                  <TooltipTrigger asChild>{buttonContent}</TooltipTrigger>
                  <TooltipContent side="right" sideOffset={12}>
                    <p className="font-semibold">{item.label}</p>
                    <p className="text-[10px] text-muted-foreground">{item.description}</p>
                  </TooltipContent>
                </Tooltip>
              );
            }

            return <div key={item.id}>{buttonContent}</div>;
          })}
        </nav>

        {/* Footer / Status & Collapse Toggle */}
        <div className="border-t p-2">
          {/* Status Indicator */}
          <div
            className={cn(
              'flex items-center rounded-md px-2 py-1.5 text-xs text-muted-foreground mb-1',
              collapsed ? 'justify-center px-0' : 'gap-2 justify-between',
            )}
          >
            {collapsed ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="flex size-4 items-center justify-center">
                    {backendError ? (
                      <span className="size-2 rounded-full bg-destructive" />
                    ) : backendReady ? (
                      <span className="size-2 rounded-full bg-emerald-500" />
                    ) : (
                      <span className="size-2 rounded-full bg-muted-foreground animate-pulse" />
                    )}
                  </span>
                </TooltipTrigger>
                <TooltipContent side="right" sideOffset={12}>
                  {backendError ? `后端异常：${backendError}` : backendReady ? '数据后端就绪' : '正在连接后端...'}
                </TooltipContent>
              </Tooltip>
            ) : (
              <div className="flex items-center gap-2 min-w-0">
                <span
                  className={cn(
                    'size-2 shrink-0 rounded-full',
                    backendError ? 'bg-destructive' : backendReady ? 'bg-emerald-500' : 'bg-muted-foreground animate-pulse',
                  )}
                />
                <span className="truncate text-[11px]">
                  {backendError ? '服务异常' : backendReady ? '核心服务在线' : '连接中...'}
                </span>
              </div>
            )}
          </div>

          {/* Toggle Button */}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onToggleCollapsed}
            className={cn('w-full text-muted-foreground hover:text-foreground h-7', collapsed ? 'px-0' : 'justify-between px-2')}
            aria-label={collapsed ? '展开侧边栏' : '折叠侧边栏'}
          >
            {!collapsed && <span className="text-xs">收起侧栏</span>}
            {collapsed ? <ChevronRight className="size-3.5" /> : <ChevronLeft className="size-3.5" />}
          </Button>
        </div>
      </aside>
    </TooltipProvider>
  );
}
