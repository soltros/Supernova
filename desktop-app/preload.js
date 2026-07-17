const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('supernovaDesktop', {
  saveInstance: (url) => ipcRenderer.invoke('save-instance', url),
  clearInstance: () => ipcRenderer.invoke('clear-instance')
});
