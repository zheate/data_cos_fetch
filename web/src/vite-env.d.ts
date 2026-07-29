/// <reference types="vite/client" />

export {};

type DesktopRuntimeConfig = {
  apiBase: string;
  token: string;
  backendReady?: boolean;
  backendError?: string;
  logPath?: string;
};

type DesktopRuntimeBridge = {
  getConfig: () => Promise<DesktopRuntimeConfig>;
  onBackendReady?: (callback: (config: DesktopRuntimeConfig) => void) => void;
  onBackendError?: (callback: (error: string) => void) => void;
  openLogsFolder?: () => Promise<string>;
};

declare global {
  interface Window {
    desktopRuntime?: DesktopRuntimeBridge;
  }
}
