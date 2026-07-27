import * as vscode from 'vscode'
import * as path from 'path'
import * as cp from 'child_process'
import * as fs from 'fs'
import * as net from 'net'
import * as http from 'http'

let outputChannel: vscode.OutputChannel

interface DevoInstance {
  id: number
  process: cp.ChildProcess | null
  port: number | null
  panel: vscode.WebviewPanel | null
}

const instances: DevoInstance[] = []
let nextInstanceId = 1

export function activate(context: vscode.ExtensionContext) {
  outputChannel = vscode.window.createOutputChannel('Devo')
  context.subscriptions.push(outputChannel)
  outputChannel.appendLine('Devo extension activated')

  const disposable = vscode.commands.registerCommand('devo.open', async () => {
    await createNewInstance(context)
  })

  context.subscriptions.push(disposable)
}

async function createNewInstance(context: vscode.ExtensionContext) {
  const instanceId = nextInstanceId++

  const instance: DevoInstance = {
    id: instanceId,
    process: null,
    port: null,
    panel: null,
  }
  instances.push(instance)

  outputChannel.appendLine(`[instance-${instanceId}] Creating new instance...`)

  try {
    await startDevoProcess(instance, context)
  } catch (err) {
    outputChannel.appendLine(`[instance-${instanceId}] Failed to start: ${err}`)
    vscode.window.showErrorMessage(`Failed to start Devo instance ${instanceId}: ${err}`)
    removeInstance(instance)
    return
  }

  createPanel(instance, context)
}

function createPanel(instance: DevoInstance, context: vscode.ExtensionContext) {
  const instanceId = instance.id
  outputChannel.appendLine(`[instance-${instanceId}] Creating webview panel on port ${instance.port}...`)

  const panel = vscode.window.createWebviewPanel(
    `devoPanel-${instanceId}`,
    `Devo # ${instanceId}`,
    vscode.ViewColumn.Beside,
    {
      enableScripts: true,
      retainContextWhenHidden: true,
      localResourceRoots: [],
    }
  )

  panel.iconPath = vscode.Uri.joinPath(context.extensionUri, 'favicon.svg')

  panel.onDidDispose(() => {
    outputChannel.appendLine(`[instance-${instanceId}] Panel disposed`)
    cleanupInstance(instance)
  })

  instance.panel = panel
  panel.webview.html = getWebviewContent(instance.port!, instanceId)
  outputChannel.appendLine(`[instance-${instanceId}] Webview panel created successfully`)
}

function cleanupInstance(instance: DevoInstance) {
  const instanceId = instance.id
  outputChannel.appendLine(`[instance-${instanceId}] Cleaning up...`)

  if (instance.process) {
    try {
      instance.process.kill()
      outputChannel.appendLine(`[instance-${instanceId}] Process killed`)
    } catch {
      // Process may already be dead
    }
  }

  removeInstance(instance)
}

function removeInstance(instance: DevoInstance) {
  const idx = instances.indexOf(instance)
  if (idx >= 0) {
    instances.splice(idx, 1)
  }
}

function handleProcessCrash(instance: DevoInstance, code: number | null) {
  const instanceId = instance.id
  instance.process = null
  instance.port = null

  outputChannel.appendLine(`[instance-${instanceId}] Process crashed with code ${code}`)

  if (instance.panel) {
    instance.panel.webview.html = getCrashContent(code, instanceId)
  }

  vscode.window.showWarningMessage(
    `Devo # ${instanceId} process exited (code: ${code ?? 'unknown'}). Click to restart.`,
    'Restart'
  ).then(selection => {
    if (selection === 'Restart') {
      if (instance.panel) {
        instance.panel.dispose()
        instance.panel = null
      }
      removeInstance(instance)
      vscode.commands.executeCommand('devo.open')
    }
  })
}

function getWebviewContent(port: number, instanceId: number): string {
  const url = `http://127.0.0.1:${port}/?mode=vscode`

  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Devo # ${instanceId}</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    html, body { width: 100%; height: 100%; overflow: hidden; }
    iframe { width: 100%; height: 100%; border: none; }
    .loading {
      display: flex; align-items: center; justify-content: center;
      width: 100%; height: 100%; background: #1a1b26;
      color: #a9b1d6; font-family: -apple-system, BlinkMacSystemFont, sans-serif;
      font-size: 14px;
    }
    .loading-spinner {
      width: 24px; height: 24px; border: 2px solid #3b4261;
      border-top-color: #7aa2f7; border-radius: 50%;
      animation: spin 0.8s linear infinite;
      margin-right: 12px;
    }
    @keyframes spin { to { transform: rotate(360deg); } }
  </style>
</head>
<body>
  <div class="loading" id="loading">
    <div class="loading-spinner"></div>
    <span>Connecting to Devo # ${instanceId}...</span>
  </div>
  <iframe id="devo-frame" src="${url}" style="display:none;"
    onload="document.getElementById('loading').style.display='none';this.style.display='block';"
    onerror="document.getElementById('loading').innerHTML='Failed to connect to Devo server at ${url}'">
  </iframe>
</body>
</html>`
}

function getCrashContent(code: number | null, instanceId: number): string {
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
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
    }
    .crash-icon { font-size: 48px; margin-bottom: 16px; }
    .crash-title { font-size: 18px; font-weight: 600; margin-bottom: 8px; color: #f7768e; }
    .crash-message { font-size: 13px; color: #565f89; }
  </style>
</head>
<body>
  <div class="crash-icon">💥</div>
  <div class="crash-title">Devo # ${instanceId} Crashed</div>
  <div class="crash-message">Exit code: ${code ?? 'unknown'}</div>
</body>
</html>`
}

function findFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer()
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      if (address && typeof address === 'object') {
        const port = address.port
        server.close(() => resolve(port))
      } else {
        server.close(() => reject(new Error('Failed to get port')))
      }
    })
    server.on('error', reject)
  })
}

function getDevoPath(context: vscode.ExtensionContext): string {
  const platform = process.platform
  const arch = process.arch
  let binaryName: string

  if (platform === 'win32') {
    binaryName = 'devo-windows-amd64.exe'
  } else if (platform === 'linux') {
    binaryName = 'devo-linux-amd64'
  } else if (platform === 'darwin') {
    binaryName = arch === 'arm64' ? 'devo-darwin-arm64' : 'devo-darwin-amd64'
  } else {
    binaryName = 'devo'
  }

  const bundledPath = path.join(context.extensionPath, 'bin', binaryName)

  if (fs.existsSync(bundledPath)) {
    if (platform !== 'win32') {
      try {
        fs.chmodSync(bundledPath, 0o755)
      } catch {
        // chmod may fail in some edge cases, but spawn will still try
      }
    }
    return bundledPath
  }

  return 'devo'
}

function waitForHealth(port: number, timeoutMs: number): Promise<void> {
  return new Promise((resolve, reject) => {
    const url = `http://127.0.0.1:${port}/api/v1/health`
    const startTime = Date.now()

    function poll() {
      if (Date.now() - startTime > timeoutMs) {
        return reject(new Error('Devo startup timed out after 30 seconds'))
      }

      const req = http.get(url, (res) => {
        if (res.statusCode === 200) {
          res.resume()
          resolve()
        } else {
          res.resume()
          setTimeout(poll, 500)
        }
      })

      req.on('error', () => {
        setTimeout(poll, 500)
      })

      req.setTimeout(3000, () => {
        req.destroy()
        setTimeout(poll, 500)
      })
    }

    poll()
  })
}

async function startDevoProcess(
  instance: DevoInstance,
  context: vscode.ExtensionContext
): Promise<void> {
  const instanceId = instance.id
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath

  instance.port = await findFreePort()
  outputChannel.appendLine(`[instance-${instanceId}] Found free port: ${instance.port}`)

  const args = ['--port', String(instance.port)]
  if (workspaceRoot) {
    args.push('--workspace', workspaceRoot)
    outputChannel.appendLine(`[instance-${instanceId}] Workspace: ${workspaceRoot}`)
  }

  const devoPath = getDevoPath(context)
  outputChannel.appendLine(`[instance-${instanceId}] Binary path: ${devoPath}`)
  outputChannel.appendLine(`[instance-${instanceId}] Spawning: ${devoPath} ${args.join(' ')}`)

  return new Promise((resolve, reject) => {
    instance.process = cp.spawn(devoPath, args, {
      stdio: ['pipe', 'pipe', 'pipe'],
      env: { ...process.env },
    })
    outputChannel.appendLine(`[instance-${instanceId}] PID: ${instance.process.pid}`)

    instance.process.stdout?.on('data', (data: Buffer) => {
      outputChannel.appendLine(`[instance-${instanceId}][stdout] ${data.toString().trim()}`)
    })

    instance.process.stderr?.on('data', (data: Buffer) => {
      outputChannel.appendLine(`[instance-${instanceId}][stderr] ${data.toString().trim()}`)
    })

    instance.process.on('error', (err) => {
      instance.process = null
      instance.port = null
      outputChannel.appendLine(`[instance-${instanceId}] Error: ${err.message}`)
      reject(new Error(`Cannot start devo: ${err.message}.`))
    })

    instance.process.on('close', (code) => {
      outputChannel.appendLine(`[instance-${instanceId}] Process exited with code ${code}`)
      if (instance.port === null) {
        reject(new Error(`Devo exited with code ${code} before port was assigned`))
      } else {
        handleProcessCrash(instance, code)
      }
    })

    waitForHealth(instance.port!, 30000)
      .then(() => {
        outputChannel.appendLine(`[instance-${instanceId}] Server started successfully`)
        resolve()
      })
      .catch((err) => {
        outputChannel.appendLine(`[instance-${instanceId}] Startup timed out after 30 seconds`)
        reject(err)
      })
  })
}

export function deactivate() {
  outputChannel?.appendLine('[deactivate] Deactivating extension, cleaning up all instances...')
  for (const instance of [...instances]) {
    cleanupInstance(instance)
  }
}