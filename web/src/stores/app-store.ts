import { create } from 'zustand';
import { toast } from 'sonner';

type AppState = {
  apiBase: string;
  token: string;
  busy: boolean;
  message: string;
  mainTab: 'data_fetch' | 'cos_filter';
  backendReady: boolean;

  setApiBase: (v: string) => void;
  setToken: (v: string) => void;
  setBusy: (v: boolean) => void;
  setMessage: (v: string) => void;
  setMainTab: (v: 'data_fetch' | 'cos_filter') => void;
  setBackendReady: (v: boolean) => void;

  /** Run an async task with busy/message handling. */
  withTask: <T>(
    task: () => Promise<T>,
    options?: {
      loading?: string;
      success?: string | ((data: T) => string);
      error?: string | ((err: unknown) => string);
    }
  ) => Promise<T | undefined>;
};

export const useAppStore = create<AppState>((set, get) => ({
  apiBase: 'http://127.0.0.1:9002',
  token: '',
  busy: false,
  message: '',
  mainTab: 'data_fetch',
  backendReady: false,

  setApiBase: (v) => set({ apiBase: v }),
  setToken: (v) => set({ token: v }),
  setBusy: (v) => set({ busy: v }),
  setMessage: (v) => set({ message: v }),
  setMainTab: (v) => set({ mainTab: v }),
  setBackendReady: (v) => set({ backendReady: v }),

  withTask: async (task, options) => {
    if (get().busy) return;
    set({ busy: true, message: '' });
    
    try {
      const promise = task();
      
      if (options) {
        toast.promise(promise, {
          loading: options.loading || '正在处理...',
          success: options.success || '操作成功',
          error: options.error || ((err: unknown) => {
            const text = err instanceof Error ? err.message : String(err);
            return `操作失败: ${text}`;
          })
        });
      }
      
      const result = await promise;
      return result;
    } catch (error) {
      if (!options) {
        const text = error instanceof Error ? error.message : String(error);
        toast.error(`操作失败: ${text}`);
      }
      return undefined;
    } finally {
      set({ busy: false });
    }
  },
}));

