const { app, BrowserWindow, dialog } = require("electron");
const path = require("path");
const cp = require("child_process");
const fs = require("fs");
const net = require("net");

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
  const binaryName = platform === "win32" ? "devo.exe" : "devo";

  if (app.isPackaged) {
    return path.join(process.resourcesPath, "bin", binaryName);
  }

  const bundledPath = path.join(__dirname, "resources", "bin", binaryName);
  if (fs.existsSync(bundledPath)) {
    return bundledPath;
  }

  return "devo";
}

// ====================== 状态 ======================
let mainWindow = null;
let serverProcess = null;
let serverPort = null;
let isQuitting = false;

// ====================== 创建窗口 ======================
function createWindow() {
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
        startServer().then(() => {
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

// ====================== 启动 Go 后端 ======================
function startServer() {
  return new Promise(async (resolve, reject) => {
    serverPort = await findFreePort();
    log("Free port:", serverPort);

    const devoPath = getDevoPath();
    const args = ["--port", String(serverPort)];
    log("Binary path:", devoPath);
    log("Spawning:", devoPath, args.join(" "));

    serverProcess = cp.spawn(devoPath, args, {
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env },
    });
    log("PID:", serverProcess.pid);

    const timeout = setTimeout(() => {
      log("Startup timed out after 30 seconds");
      reject(new Error("Devo 启动超时（30 秒）"));
    }, 30000);

    serverProcess.stdout.on("data", (data) => {
      const text = data.toString();
      console.log(`[devo][stdout] ${text.trim()}`);
      if (text.includes("Server ready")) {
        clearTimeout(timeout);
        log("Server started successfully");
        resolve();
      }
    });

    serverProcess.stderr.on("data", (data) => {
      const text = data.toString();
      console.log(`[devo][stderr] ${text.trim()}`);
      if (text.includes("Server ready")) {
        clearTimeout(timeout);
        log("Server started successfully (via stderr)");
        resolve();
      }
    });

    serverProcess.on("error", (err) => {
      clearTimeout(timeout);
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
      clearTimeout(timeout);
      log("Process exited with code:", code);
      if (serverPort === null) {
        reject(new Error(`Devo 在端口分配前退出（退出码: ${code}）`));
      } else if (!isQuitting) {
        handleProcessCrash(code);
      }
    });
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

// ====================== 应用生命周期 ======================
app.whenReady().then(async () => {
  log("App starting...");

  try {
    await startServer();
    createWindow();
  } catch (err) {
    log("Failed to start:", err.message);
    app.quit();
  }
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
  if (mainWindow === null) {
    createWindow();
  }
});