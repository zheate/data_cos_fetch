import { lazy, Suspense, useEffect } from 'react';
import { AppNavbar } from './components/AppNavbar';
import { useAppStore } from './stores/app-store';
import { Skeleton } from '@/components/ui/skeleton';

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

declare global {
  interface Window {
    desktopRuntime?: {
      getConfig: () => Promise<{ apiBase: string; token: string; backendReady?: boolean }>;
      onBackendReady?: (callback: (config: { apiBase: string; token: string; backendReady?: boolean }) => void) => void;
      onBackendError?: (callback: (error: string) => void) => void;
    };
  }
}

function App() {
  const { mainTab, setApiBase, setToken, setBackendReady } = useAppStore();
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
        setBackendReady(true);
      });
    }

    if (runtime.onBackendError) {
      runtime.onBackendError((error) => {
        console.error('backend startup failed:', error);
        // Still mark ready so the user can see the UI (with degraded backend).
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
        if (config.backendReady) setBackendReady(true);
      })
      .catch(() => {
        // Running in browser mode — use defaults
      });
  }, [setApiBase, setToken, setBackendReady]);

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
    <div className="app-shell min-h-dvh bg-background text-foreground antialiased selection:bg-primary/15">
      <AppNavbar />

      <main className="mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-4 px-4 py-6 sm:px-6 lg:px-8">
        {/* Removed global Alert banner, relying on sonner toast instead */}
        <Suspense fallback={<ViewFallback />}>
          {mainTab === 'data_fetch' ? <DataFetchView /> : <CosFilterView />}
        </Suspense>
      </main>
    </div>
  );
}

export default App;
