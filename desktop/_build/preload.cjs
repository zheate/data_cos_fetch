const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('desktopRuntime', {
  getConfig: async () => ipcRenderer.invoke('desktop:get-runtime-config'),
});
