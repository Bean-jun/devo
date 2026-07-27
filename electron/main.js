const { app, BrowserWindow, dialog, ipcMain } = require("electron");
const path = require("path");
const cp = require("child_process");
const fs = require("fs");
const net = require("net");
const http = require("http");

// ====================== 日志 ======================
function log(...args) {
  console.log(`[devo-desktop]`, ...args);
}

// ====================== 端口分配 ======================
function findFreePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.listen(0, "127.0.0.1", () => {
      const addr = server.address();
      if (addr && typeof addr === "object") {
        const port = addr.port;
        server.close(() => resolve(port));
      } else {
        server.close(() => reject(new Error("Failed to get port")));
      }
    });
    server.on("error", reject);
  });
}

// ====================== Go 二进制路径 ======================
function getDevoPath() {
  const platform = process.platform;
  const arch = process.arch;
  let binaryName;

  if (platform === "win32") {
    binaryName = "devo-windows-amd64.exe";
  } else if (platform === "linux") {
    binaryName = "devo-linux-amd64";
  } else if (platform === "darwin") {
    binaryName = arch === "arm64" ? "devo-darwin-arm64" : "devo-darwin-amd64";
  } else {
    binaryName = "devo";
  }

  const bundledPath = app.isPackaged
    ? path.join(process.resourcesPath, "bin", binaryName)
    : path.join(__dirname, "resources", "bin", binaryName);

  if (fs.existsSync(bundledPath)) {
    if (platform !== "win32") {
      try { fs.chmodSync(bundledPath, 0o755); } catch {}
    }
    return bundledPath;
  }

  return "devo";
}

// ====================== 状态 ======================
let welcomeWindow = null;
let mainWindow = null;
let serverProcess = null;
let serverPort = null;
let currentWorkspace = null;
let isQuitting = false;

// ====================== 欢迎页窗口 ======================
function createWelcomeWindow() {
  welcomeWindow = new BrowserWindow({
    width: 900,
    height: 680,
    minWidth: 600,
    minHeight: 480,
    resizable: true,
    autoHideMenuBar: true,
    title: "Devo",
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false,
    },
  });

  welcomeWindow.loadFile(path.join(__dirname, "welcome.html"));

  welcomeWindow.on("closed", () => {
    welcomeWindow = null;
    if (!mainWindow) {
      app.quit();
    }
  });
}

// ====================== 主窗口 ======================
function createMainWindow() {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 900,
    minHeight: 600,
    autoHideMenuBar: true,
    title: "Devo",
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
    },
  });

  const url = `http://127.0.0.1:${serverPort}/`;
  log("Loading URL:", url);
  mainWindow.loadURL(url);

  mainWindow.on("closed", () => {
    mainWindow = null;
  });
}

// ====================== 崩溃恢复 ======================
function handleProcessCrash(code) {
  log("Go process crashed with code:", code);
  serverProcess = null;
  serverPort = null;

  if (mainWindow) {
    mainWindow.loadURL(
      `data:text/html,${encodeURIComponent(getCrashContent(code))}`
    );
  }

  dialog
    .showMessageBox({
      type: "error",
      title: "Devo 后端已退出",
      message: `Devo 后端进程异常退出（退出码: ${code ?? "unknown"}）。`,
      detail: "是否重新启动？",
      buttons: ["重新启动", "退出"],
      defaultId: 0,
    })
    .then((result) => {
      if (result.response === 0) {
        log("User chose to restart");
        startServer(currentWorkspace).then(() => {
          if (mainWindow) {
            mainWindow.loadURL(`http://127.0.0.1:${serverPort}/`);
          }
        });
      } else {
        log("User chose to quit");
        app.quit();
      }
    });
}

function getCrashContent(code) {
  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    html, body { width: 100%; height: 100%; }
    body {
      display: flex; flex-direction: column;
      align-items: center; justify-content: center;
      background: #1a1b26; color: #a9b1d6;
      font-family: -apple-system, BlinkMacSystemFont, "Microsoft YaHei", sans-serif;
    }
    .crash-icon { font-size: 48px; margin-bottom: 16px; }
    .crash-title { font-size: 18px; font-weight: 600; margin-bottom: 8px; color: #f7768e; }
    .crash-message { font-size: 13px; color: #565f89; }
  </style>
</head>
<body>
  <div class="crash-icon">💥</div>
  <div class="crash-title">Devo 后端已退出</div>
  <div class="crash-message">退出码: ${code ?? "unknown"}</div>
</body>
</html>`;
}

// ====================== 健康检查轮询 ======================
function waitForHealth(port, timeoutMs) {
  return new Promise((resolve, reject) => {
    const url = `http://127.0.0.1:${port}/api/v1/health`;
    const startTime = Date.now();

    function poll() {
      if (Date.now() - startTime > timeoutMs) {
        return reject(new Error("Devo 启动超时（30 秒）"));
      }

      const req = http.get(url, (res) => {
        if (res.statusCode === 200) {
          res.resume();
          resolve();
        } else {
          res.resume();
          setTimeout(poll, 500);
        }
      });

      req.on("error", () => {
        setTimeout(poll, 500);
      });

      req.setTimeout(3000, () => {
        req.destroy();
        setTimeout(poll, 500);
      });
    }

    poll();
  });
}

// ====================== 启动 Go 后端 ======================
function startServer(workspace) {
  return new Promise(async (resolve, reject) => {
    currentWorkspace = workspace;
    serverPort = await findFreePort();
    log("Free port:", serverPort);

    const devoPath = getDevoPath();
    const args = ["--port", String(serverPort), "--workspace", workspace];
    log("Binary path:", devoPath);
    log("Workspace:", workspace);
    log("Spawning:", devoPath, args.join(" "));

    serverProcess = cp.spawn(devoPath, args, {
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env },
    });
    log("PID:", serverProcess.pid);

    serverProcess.stdout.on("data", (data) => {
      console.log(`[devo][stdout] ${data.toString().trim()}`);
    });

    serverProcess.stderr.on("data", (data) => {
      console.log(`[devo][stderr] ${data.toString().trim()}`);
    });

    serverProcess.on("error", (err) => {
      serverProcess = null;
      serverPort = null;
      log("Process error:", err.message);
      dialog.showErrorBox(
        "启动失败",
        `无法启动 Devo 后端：${err.message}\n\n请确保 devo 已安装并在 PATH 中，或已编译到 resources/bin/ 目录。`
      );
      reject(err);
    });

    serverProcess.on("close", (code) => {
      log("Process exited with code:", code);
      if (serverPort === null) {
        reject(new Error(`Devo 在端口分配前退出（退出码: ${code}）`));
      } else if (!isQuitting) {
        handleProcessCrash(code);
      }
    });

    try {
      await waitForHealth(serverPort, 30000);
      log("Server started successfully");
      resolve();
    } catch (err) {
      reject(err);
    }
  });
}

// ====================== 清理 ======================
function cleanupServer() {
  if (serverProcess) {
    log("Killing server process...");
    try {
      serverProcess.kill();
    } catch {
      // Process may already be dead
    }
    serverProcess = null;
  }
}

// ====================== IPC 处理 ======================
function setupIPC() {
  ipcMain.on("select-folder", async (event) => {
    const result = await dialog.showOpenDialog(welcomeWindow, {
      properties: ["openDirectory"],
      title: "选择项目文件夹",
    });

    if (!result.canceled && result.filePaths.length > 0) {
      event.reply("folder-selected", result.filePaths[0]);
    }
  });

  ipcMain.on("open-recent", async (_event, folderPath) => {
    log("Opening recent folder:", folderPath);

    if (!fs.existsSync(folderPath)) {
      dialog.showErrorBox(
        "路径不存在",
        `所选目录不存在：\n${folderPath}\n\n该目录可能已被移动或删除。`
      );
      return;
    }

    try {
      await startServer(folderPath);

      if (welcomeWindow) {
        welcomeWindow.close();
      }
      createMainWindow();
    } catch (err) {
      log("Failed to start server:", err.message);
    }
  });
}

// ====================== 应用生命周期 ======================
app.whenReady().then(() => {
  log("App starting...");
  setupIPC();
  createWelcomeWindow();
});

app.on("window-all-closed", () => {
  log("All windows closed");
  isQuitting = true;
  cleanupServer();
  app.quit();
});

app.on("before-quit", () => {
  isQuitting = true;
  cleanupServer();
});

app.on("activate", () => {
  if (mainWindow === null && welcomeWindow === null) {
    createWelcomeWindow();
  }
});