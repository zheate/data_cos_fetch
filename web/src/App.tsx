import { lazy, Suspense, useEffect, useState } from 'react';
import { AlertTriangle, FolderOpen } from 'lucide-react';
import { AppSidebar } from './components/layout/AppSidebar';
import { AppHeader } from './components/layout/AppHeader';
import { useAppStore } from './stores/app-store';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { injectAllMockData } from './helpers/mockData';

const loadDataFetchView = () =>
  import('./views/DataFetchView').then((module) => ({ default: module.DataFetchView }));
const loadCosFilterView = () =>
  import('./views/CosFilterView').then((module) => ({ default: module.CosFilterView }));

const DataFetchView = lazy(loadDataFetchView);
const CosFilterView = lazy(loadCosFilterView);

function ViewFallback() {
  return (
    <div className="grid gap-6 lg:grid-cols-[340px_minmax(0,1fr)] xl:grid-cols-[380px_minmax(0,1fr)] opacity-70">
      <div className="flex flex-col gap-6">
        <Skeleton className="h-[480px] w-full rounded-xl" />
      </div>
      <div className="flex flex-col gap-4">
        <div className="grid gap-4 md:grid-cols-3">
          <Skeleton className="h-28 w-full rounded-xl" />
          <Skeleton className="h-28 w-full rounded-xl" />
          <Skeleton className="h-28 w-full rounded-xl" />
        </div>
        <Skeleton className="h-[680px] w-full rounded-xl" />
      </div>
    </div>
  );
}

function App() {
  const {
    mainTab,
    backendError,
    logPath,
    setApiBase,
    setToken,
    setBackendReady,
    setBackendError,
    setLogPath,
  } = useAppStore();

  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem('cos_sidebar_collapsed') === 'true',
  );

  const toggleSidebar = () => {
    setCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem('cos_sidebar_collapsed', String(next));
      return next;
    });
  };

  // Preload realistic mock data for UI and layout evaluation
  useEffect(() => {
    injectAllMockData();
  }, []);

  // Listen for the push-based backend-ready IPC event (new path).
  // Falls back to the pull-based getConfig() for browser-only mode.
  useEffect(() => {
    const runtime = window.desktopRuntime;
    if (!runtime) {
      // Running in browser — no Electron, mark ready immediately.
      setBackendReady(true);
      return;
    }

    // Push path: main process sends config when backend passes health check.
    if (runtime.onBackendReady) {
      runtime.onBackendReady((config) => {
        if (config.apiBase) setApiBase(config.apiBase);
        if (config.token) setToken(config.token);
        if (config.logPath) setLogPath(config.logPath);
        setBackendError('');
        setBackendReady(true);
      });
    }

    if (runtime.onBackendError) {
      runtime.onBackendError((error) => {
        console.error('backend startup failed:', error);
        setBackendError(error);
        setBackendReady(true);
      });
    }

    // Pull path: in case the backend was already ready before the renderer
    // registered its listener (race condition on fast machines).
    runtime
      .getConfig()
      .then((config) => {
        if (config.apiBase) setApiBase(config.apiBase);
        if (config.token) setToken(config.token);
        if (config.logPath) setLogPath(config.logPath);
        if (config.backendError) setBackendError(config.backendError);
        if (config.backendReady) setBackendReady(true);
      })
      .catch(() => {
        // Running in browser mode — use defaults
      });
  }, [setApiBase, setToken, setBackendReady, setBackendError, setLogPath]);

  const openLogsFolder = async () => {
    const error = await window.desktopRuntime?.openLogsFolder?.();
    if (error) setBackendError(`${backendError}\n打开日志目录失败：${error}`);
  };

  // Auto-dismiss messages effect is no longer needed since we use sonner toast
  // Remove unused timer

  useEffect(() => {
    const timer = window.setTimeout(() => {
      if (mainTab === 'data_fetch') {
        void loadCosFilterView();
      } else {
        void loadDataFetchView();
      }
    }, 1200);

    return () => window.clearTimeout(timer);
  }, [mainTab]);

  return (
    <div className="app-shell flex min-h-dvh bg-background text-foreground antialiased selection:bg-primary/15">
      <AppSidebar collapsed={collapsed} onToggleCollapsed={toggleSidebar} />

      <div className="flex min-w-0 flex-1 flex-col">
        <AppHeader />

        <main className="flex-1 flex flex-col gap-4 p-4 sm:p-5 lg:p-6 w-full">
          {backendError && (
            <div role="alert" className="flex flex-col gap-3 rounded-lg border border-destructive/35 bg-destructive/5 px-4 py-3 text-sm sm:flex-row sm:items-start sm:justify-between">
              <div className="flex min-w-0 gap-3">
                <AlertTriangle className="mt-0.5 size-4 shrink-0 text-destructive" />
                <div className="min-w-0">
                  <p className="font-semibold text-destructive">数据后端启动失败</p>
                  <p className="mt-1 break-words font-mono text-xs text-muted-foreground">{backendError}</p>
                  {logPath && <p className="mt-1 break-all text-xs text-muted-foreground">日志：{logPath}</p>}
                </div>
              </div>
              {window.desktopRuntime?.openLogsFolder && (
                <Button type="button" variant="outline" size="sm" className="shrink-0" onClick={openLogsFolder}>
                  <FolderOpen data-icon="inline-start" />
                  打开日志目录
                </Button>
              )}
            </div>
          )}
          <Suspense fallback={<ViewFallback />}>
            {mainTab === 'data_fetch' ? <DataFetchView /> : <CosFilterView />}
          </Suspense>
        </main>
      </div>
    </div>
  );
}

export default App;
