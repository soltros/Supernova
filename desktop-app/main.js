const { app, BrowserWindow, ipcMain, Menu } = require('electron');
const path = require('path');
const fs = require('fs');
const { pathToFileURL } = require('url');
const { normalizeInstanceUrl, sameOrigin } = require('./security');
const SETUP_PATH = path.join(__dirname, 'setup.html');
const SETUP_URL = pathToFileURL(SETUP_PATH).href;

const CONFIG_PATH = path.join(app.getPath('userData'), 'config.json');

function getConfig() {
  try {
    if (fs.existsSync(CONFIG_PATH)) {
      const data = fs.readFileSync(CONFIG_PATH, 'utf8');
      const config = JSON.parse(data);
      return { instanceUrl: config.instanceUrl ? normalizeInstanceUrl(config.instanceUrl) : null };
    }
  } catch (e) {
    console.error('Failed to read config', e);
  }
  return { instanceUrl: null };
}

function saveConfig(config) {
  try {
    fs.mkdirSync(path.dirname(CONFIG_PATH), { recursive: true });
    fs.writeFileSync(CONFIG_PATH, JSON.stringify(config), { mode: 0o600 });
  } catch (e) {
    console.error('Failed to save config', e);
    throw e;
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
      contextIsolation: true,
      sandbox: true
    },
    autoHideMenuBar: true,
  });

  mainWindow.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
  mainWindow.webContents.on('will-navigate', (event, url) => {
    const instance = getConfig().instanceUrl;
    if (url !== SETUP_URL && !sameOrigin(url, instance)) event.preventDefault();
  });
  mainWindow.webContents.session.setPermissionRequestHandler((_contents, _permission, callback) => callback(false));
  const config = getConfig();

  if (config.instanceUrl) {
    mainWindow.loadURL(config.instanceUrl).catch(() => {
      // If it fails to load the URL, fallback to setup
      mainWindow.loadFile(SETUP_PATH);
    });
  } else {
    mainWindow.loadFile(SETUP_PATH);
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
  if (!mainWindow || event.sender !== mainWindow.webContents || event.senderFrame !== mainWindow.webContents.mainFrame || event.senderFrame.url !== SETUP_URL) throw new Error('Only setup can change the server.');
  url = normalizeInstanceUrl(url);

  saveConfig({ instanceUrl: url });
  
  // Reload with new URL
  if (mainWindow) {
    mainWindow.loadURL(url).catch(() => {
      mainWindow.loadFile(SETUP_PATH);
    });
  }
  return true;
});

ipcMain.handle('clear-instance', (event) => {
  if (!mainWindow || event.sender !== mainWindow.webContents || event.senderFrame !== mainWindow.webContents.mainFrame || (event.senderFrame.url !== SETUP_URL && !sameOrigin(event.senderFrame.url, getConfig().instanceUrl))) throw new Error('Untrusted request.');
  saveConfig({ instanceUrl: null });
  if (mainWindow) {
    mainWindow.loadFile(SETUP_PATH);
  }
  return true;
});


function trustedMainRenderer(event) {
  const instance = getConfig().instanceUrl;
  return !!mainWindow &&
    event.sender === mainWindow.webContents &&
    event.senderFrame === mainWindow.webContents.mainFrame &&
    sameOrigin(event.senderFrame.url, instance);
}

function isLastFmAuthUrl(value) {
  try {
    const url = new URL(value);
    return url.protocol === 'https:' &&
      (url.hostname === 'www.last.fm' || url.hostname === 'last.fm') &&
      (url.pathname === '/api/auth' || url.pathname === '/api/auth/');
  } catch {
    return false;
  }
}

ipcMain.handle('open-lastfm-auth', async (event, authUrl) => {
  if (!trustedMainRenderer(event)) throw new Error('Untrusted request.');
  if (!isLastFmAuthUrl(authUrl)) throw new Error('Invalid Last.fm authorization URL.');

  const instance = getConfig().instanceUrl;
  const oauthWindow = new BrowserWindow({
    parent: mainWindow,
    modal: true,
    width: 720,
    height: 800,
    title: 'Connect Last.fm',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: true
    }
  });
  oauthWindow.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
  oauthWindow.webContents.session.setPermissionRequestHandler((_contents, _permission, callback) => callback(false));

  const handleNavigation = (event, target) => {
    if (isLastFmAuthUrl(target)) return;
    try {
      const url = new URL(target);
      const callbackAllowed = sameOrigin(target, instance) && url.pathname === '/settings' && url.searchParams.has('token');
      const lastFmAllowed = url.protocol === 'https:' && (url.hostname === 'www.last.fm' || url.hostname === 'last.fm');
      if (callbackAllowed) {
        event.preventDefault();
        void mainWindow.loadURL(target);
        oauthWindow.close();
        return;
      }
      if (lastFmAllowed) return;
    } catch {}
    event.preventDefault();
  };
  oauthWindow.webContents.on('will-navigate', handleNavigation);
  await oauthWindow.loadURL(authUrl);
  return true;
});