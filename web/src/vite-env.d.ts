/// <reference types="vite/client" />

export {};

type DesktopRuntimeConfig = {
  apiBase: string;
  token: string;
};

type DesktopRuntimeBridge = {
  getConfig: () => Promise<DesktopRuntimeConfig>;
  onBackendReady?: (callback: (config: DesktopRuntimeConfig) => void) => void;
  onBackendError?: (callback: (error: string) => void) => void;
};

declare global {
  interface Window {
    desktopRuntime?: DesktopRuntimeBridge;
  }
}
