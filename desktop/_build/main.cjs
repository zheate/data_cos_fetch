const { app, BrowserWindow, ipcMain } = require('electron');
const crypto = require('node:crypto');
const http = require('node:http');
const net = require('node:net');
const path = require('node:path');
const { spawn } = require('node:child_process');

let backendProcess = null;
let runtimeConfig = {
  apiBase: 'http://127.0.0.1:9002',
  token: '',
  backendReady: false,
};

function randomToken() {
  return crypto.randomBytes(24).toString('hex');
}

function getFreePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.unref();
    server.on('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      if (!address || typeof address === 'string') {
        reject(new Error('failed to resolve free port'));
        return;
      }
      const { port } = address;
      server.close((err) => {
        if (err) {
          reject(err);
          return;
        }
        resolve(port);
      });
    });
  });
}

function waitForHealth(apiBase, timeoutMs = 20000) {
  const startedAt = Date.now();

  return new Promise((resolve, reject) => {
    const check = () => {
      if (Date.now() - startedAt > timeoutMs) {
        reject(new Error('rust api health check timeout'));
        return;
      }

      const req = http.request(`${apiBase}/health`, { method: 'GET', timeout: 1000 }, (res) => {
        if (res.statusCode === 200) {
          resolve();
        } else {
          setTimeout(check, 250);
        }
      });

      req.on('error', () => {
        setTimeout(check, 250);
      });
      req.on('timeout', () => {
        req.destroy();
        setTimeout(check, 250);
      });
      req.end();
    };

    check();
  });
}

function waitForEarlyBackendExit(proc) {
  let cleanup = () => {};

  const promise = new Promise((_, reject) => {
    const onError = (error) => {
      cleanup();
      reject(new Error(`failed to start rust api: ${error.message}`));
    };
    const onExit = (code, signal) => {
      cleanup();
      reject(new Error(`rust api exited before health check (code=${code}, signal=${signal})`));
    };

    cleanup = () => {
      proc.off('error', onError);
      proc.off('exit', onExit);
    };

    proc.once('error', onError);
    proc.once('exit', onExit);
  });

  return { promise, cleanup };
}

function resolveBackendCommand() {
  const isPackaged = app.isPackaged;
  if (isPackaged) {
    const binaryName = process.platform === 'win32' ? 'data-cos-api.exe' : 'data-cos-api';
    const binaryPath = path.join(process.resourcesPath, 'bin', binaryName);
    return {
      command: binaryPath,
      args: [],
      cwd: path.dirname(binaryPath),
    };
  }

  const rustRoot = path.resolve(__dirname, '../rust');
  return {
    command: 'cargo',
    args: ['run', '--manifest-path', path.join(rustRoot, 'Cargo.toml'), '-p', 'data-cos-api'],
    cwd: rustRoot,
  };
}

async function startBackend() {
  if (backendProcess) {
    return runtimeConfig;
  }

  const port = await getFreePort();
  const token = randomToken();
  const apiBase = `http://127.0.0.1:${port}`;

  const { command, args, cwd } = resolveBackendCommand();
  const workspaceRoot = app.isPackaged
    ? process.cwd()
    : path.resolve(__dirname, '../..');
  const defaultBatchDir = path.join(workspaceRoot, 'data');

  const env = {
    ...process.env,
    DATA_COS_API_PORT: String(port),
    DATA_COS_API_TOKEN: token,
    DATA_COS_BATCH_DIR: process.env.DATA_COS_BATCH_DIR || defaultBatchDir,
  };

  backendProcess = spawn(command, args, {
    cwd,
    env,
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  backendProcess.stdout?.on('data', (chunk) => {
    process.stdout.write(`[rust-api] ${chunk}`);
  });
  backendProcess.stderr?.on('data', (chunk) => {
    process.stderr.write(`[rust-api] ${chunk}`);
  });

  backendProcess.on('exit', (code, signal) => {
    console.log(`rust api exited (code=${code}, signal=${signal})`);
    backendProcess = null;
  });

  runtimeConfig = { apiBase, token, backendReady: false };

  const earlyExit = waitForEarlyBackendExit(backendProcess);
  try {
    await Promise.race([waitForHealth(apiBase), earlyExit.promise]);
  } finally {
    earlyExit.cleanup();
  }

  runtimeConfig = { apiBase, token, backendReady: true };
  return runtimeConfig;
}

function stopBackend() {
  if (!backendProcess) {
    return;
  }

  const proc = backendProcess;
  backendProcess = null;

  if (process.platform === 'win32') {
    spawn('taskkill', ['/pid', String(proc.pid), '/f', '/t']);
  } else {
    proc.kill('SIGTERM');
    setTimeout(() => {
      try {
        proc.kill('SIGKILL');
      } catch {
        // no-op
      }
    }, 3000);
  }
}

function createWindow() {
  const mainWindow = new BrowserWindow({
    width: 1400,
    height: 920,
    autoHideMenuBar: true,
    show: false,
    backgroundColor: '#09090b',
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });

  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  const rendererUrl = process.env.ELECTRON_RENDERER_URL;
  if (rendererUrl) {
    mainWindow.loadURL(rendererUrl);
  } else {
    const indexPath = app.isPackaged
      ? path.join(process.resourcesPath, 'web-dist', 'index.html')
      : path.resolve(__dirname, '../web/dist/index.html');
    mainWindow.loadFile(indexPath);
  }

  return mainWindow;
}

ipcMain.handle('desktop:get-runtime-config', async () => runtimeConfig);

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', () => {
  stopBackend();
});

app.whenReady().then(async () => {
  const mainWindow = createWindow();

  try {
    await startBackend();
    mainWindow.webContents.send('desktop:backend-ready', runtimeConfig);
  } catch (error) {
    console.error('failed to start backend:', error);
    mainWindow.webContents.send('desktop:backend-error', String(error));
  }

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});
