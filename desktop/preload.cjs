const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('desktopRuntime', {
  getConfig: async () => ipcRenderer.invoke('desktop:get-runtime-config'),
  onBackendReady: (callback) => {
    ipcRenderer.on('desktop:backend-ready', (_event, config) => callback(config));
  },
  onBackendError: (callback) => {
    ipcRenderer.on('desktop:backend-error', (_event, error) => callback(error));
  },
});
