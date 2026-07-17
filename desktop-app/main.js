const { app, BrowserWindow, ipcMain, Menu } = require('electron');
const path = require('path');
const fs = require('fs');

const CONFIG_PATH = path.join(app.getPath('userData'), 'config.json');

function getConfig() {
  try {
    if (fs.existsSync(CONFIG_PATH)) {
      const data = fs.readFileSync(CONFIG_PATH, 'utf8');
      return JSON.parse(data);
    }
  } catch (e) {
    console.error('Failed to read config', e);
  }
  return { instanceUrl: null };
}

function saveConfig(config) {
  try {
    fs.writeFileSync(CONFIG_PATH, JSON.stringify(config));
  } catch (e) {
    console.error('Failed to save config', e);
  }
}

let mainWindow;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    minWidth: 800,
    minHeight: 600,
    title: 'Supernova',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true
    },
    autoHideMenuBar: true,
  });

  const config = getConfig();

  if (config.instanceUrl) {
    mainWindow.loadURL(config.instanceUrl).catch(() => {
      // If it fails to load the URL, fallback to setup
      mainWindow.loadFile('setup.html');
    });
  } else {
    mainWindow.loadFile('setup.html');
  }
  
  // Clean up Menu
  Menu.setApplicationMenu(null);
}

app.whenReady().then(() => {
  createWindow();

  app.on('activate', function () {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', function () {
  if (process.platform !== 'darwin') app.quit();
});

// IPC handlers
ipcMain.handle('save-instance', (event, url) => {
  // Ensure it has http/https
  if (!/^https?:\/\//i.test(url)) {
    url = 'http://' + url;
  }
  
  // Trim trailing slash
  url = url.replace(/\/$/, "");

  saveConfig({ instanceUrl: url });
  
  // Reload with new URL
  if (mainWindow) {
    mainWindow.loadURL(url).catch(() => {
      mainWindow.loadFile('setup.html');
    });
  }
  return true;
});

ipcMain.handle('clear-instance', (event) => {
  saveConfig({ instanceUrl: null });
  if (mainWindow) {
    mainWindow.loadFile('setup.html');
  }
  return true;
});
