import * as vscode from 'vscode'
import * as path from 'path'
import * as cp from 'child_process'
import * as fs from 'fs'
import * as net from 'net'

let statusBarItem: vscode.StatusBarItem
let devoProcess: cp.ChildProcess | null = null
let devoPort: number | null = null
let panel: vscode.WebviewPanel | null = null
let outputChannel: vscode.OutputChannel

export function activate(context: vscode.ExtensionContext) {
  outputChannel = vscode.window.createOutputChannel('Devo')
  context.subscriptions.push(outputChannel)
  outputChannel.appendLine('Devo extension activated')

  statusBarItem = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Right,
    100
  )
  statusBarItem.command = 'devo.open'
  statusBarItem.text = '$(hubot) Devo'
  statusBarItem.tooltip = 'Open Devo AI Agent'
  context.subscriptions.push(statusBarItem)
  statusBarItem.show()

  const disposable = vscode.commands.registerCommand('devo.open', async () => {
    await openDevoPanel(context)
  })

  context.subscriptions.push(disposable)
}

async function openDevoPanel(context: vscode.ExtensionContext) {
  if (panel) {
    outputChannel.appendLine('[open] Panel already exists, revealing...')
    panel.reveal(vscode.ViewColumn.Beside)
    return
  }

  if (!devoProcess) {
    try {
      outputChannel.appendLine('[open] Starting Devo process...')
      await startDevoProcess(context)
    } catch (err) {
      outputChannel.appendLine(`[open] Failed to start Devo: ${err}`)
      vscode.window.showErrorMessage(`Failed to start Devo: ${err}`)
      return
    }
  }

  outputChannel.appendLine(`[open] Creating webview panel on port ${devoPort}...`)
  panel = vscode.window.createWebviewPanel(
    'devoPanel',
    'Devo',
    vscode.ViewColumn.Beside,
    {
      enableScripts: true,
      retainContextWhenHidden: true,
      localResourceRoots: [],
    }
  )

  panel.iconPath = vscode.Uri.joinPath(
    context.extensionUri,
    'favicon.svg'
  )

  panel.onDidDispose(() => {
    outputChannel.appendLine('[open] Panel disposed')
    panel = null
  })

  updateStatusBar()

  panel.webview.html = getWebviewContent(devoPort!)
  outputChannel.appendLine('[open] Webview panel created successfully')
}

function getWebviewContent(port: number): string {
  const url = `http://127.0.0.1:${port}`

  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Devo</title>
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
    <span>Connecting to Devo...</span>
  </div>
  <iframe id="devo-frame" src="${url}" style="display:none;"
    onload="document.getElementById('loading').style.display='none';this.style.display='block';"
    onerror="document.getElementById('loading').innerHTML='Failed to connect to Devo server at ${url}'">
  </iframe>
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
  const binaryName = platform === 'win32' ? 'devo.exe' : 'devo'
  const bundledPath = path.join(context.extensionPath, 'bin', binaryName)

  if (fs.existsSync(bundledPath)) {
    return bundledPath
  }

  return 'devo'
}

async function startDevoProcess(context: vscode.ExtensionContext): Promise<void> {
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath

  devoPort = await findFreePort()
  outputChannel.appendLine(`[process] Found free port: ${devoPort}`)

  const args = ['--port', String(devoPort)]
  if (workspaceRoot) {
    args.push('--workspace', workspaceRoot)
    outputChannel.appendLine(`[process] Workspace: ${workspaceRoot}`)
  }

  const devoPath = getDevoPath(context)
  outputChannel.appendLine(`[process] Binary path: ${devoPath}`)
  outputChannel.appendLine(`[process] Spawning: ${devoPath} ${args.join(' ')}`)

  return new Promise((resolve, reject) => {
    devoProcess = cp.spawn(devoPath, args, {
      stdio: ['pipe', 'pipe', 'pipe'],
      env: { ...process.env },
    })
    outputChannel.appendLine(`[process] PID: ${devoProcess.pid}`)

    const timeout = setTimeout(() => {
      outputChannel.appendLine('[process] Startup timed out after 30 seconds')
      reject(new Error('Devo startup timed out after 30 seconds'))
    }, 30000)

    devoProcess.stdout?.on('data', (data: Buffer) => {
      const text = data.toString()
      outputChannel.appendLine(`[stdout] ${text.trim()}`)
      if (text.includes('Server ready')) {
        clearTimeout(timeout)
        outputChannel.appendLine('[process] Server started successfully')
        resolve()
      }
    })

    devoProcess.stderr?.on('data', (data: Buffer) => {
      const text = data.toString()
      outputChannel.appendLine(`[stderr] ${text.trim()}`)
      if (text.includes('Server ready')) {
        clearTimeout(timeout)
        outputChannel.appendLine('[process] Server started successfully (via stderr)')
        resolve()
      }
    })

    devoProcess.on('error', (err) => {
      clearTimeout(timeout)
      devoProcess = null
      devoPort = null
      outputChannel.appendLine(`[process] Error: ${err.message}`)
      reject(new Error(`Cannot start devo: ${err.message}.`))
    })

    devoProcess.on('close', (code) => {
      clearTimeout(timeout)
      outputChannel.appendLine(`[process] Process exited with code ${code}`)
      if (devoPort === null) {
        reject(new Error(`Devo exited with code ${code} before port was assigned`))
      } else {
        handleProcessCrash(code)
      }
    })
  })
}

function handleProcessCrash(code: number | null) {
  devoProcess = null
  devoPort = null

  outputChannel.appendLine(`[crash] Devo process crashed with code ${code}`)

  updateStatusBar()

  if (panel) {
    panel.webview.html = getCrashContent(code)
  }

  vscode.window.showWarningMessage(
    `Devo process exited (code: ${code ?? 'unknown'}). Click to restart.`,
    'Restart'
  ).then(selection => {
    if (selection === 'Restart') {
      if (panel) {
        panel.dispose()
        panel = null
      }
      vscode.commands.executeCommand('devo.open')
    }
  })
}

function getCrashContent(code: number | null): string {
  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    html, body { width: 100%; height: 100%; overflow: hidden; }
    body {
      display: flex; flex-direction: column; align-items: center; justify-content: center;
      background: #1a1b26; color: #a9b1d6;
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
    }
    .icon { font-size: 48px; margin-bottom: 16px; }
    .title { font-size: 18px; font-weight: 600; color: #c0caf5; margin-bottom: 8px; }
    .detail { font-size: 13px; color: #565f89; margin-bottom: 24px; }
    .hint { font-size: 12px; color: #565f89; }
    .hint kbd {
      background: #3b4261; color: #a9b1d6; padding: 2px 6px;
      border-radius: 3px; font-family: inherit; font-size: 11px;
    }
  </style>
</head>
<body>
  <div class="icon">⚠️</div>
  <div class="title">Devo 进程已退出</div>
  <div class="detail">退出码: ${code ?? 'unknown'}</div>
  <div class="hint">点击状态栏 <kbd>$(vm-active) Devo</kbd> 重新启动</div>
</body>
</html>`
}

function updateStatusBar() {
  if (devoProcess) {
    statusBarItem.text = '$(vm-active) Devo'
    statusBarItem.tooltip = `Devo running on port ${devoPort}`
  } else {
    statusBarItem.text = '$(hubot) Devo'
    statusBarItem.tooltip = 'Open Devo AI Agent'
  }
}

export function deactivate() {
  outputChannel.appendLine('[deactivate] Devo extension deactivating...')
  if (devoProcess) {
    devoProcess.kill()
    devoProcess = null
    outputChannel.appendLine('[deactivate] Process killed')
  }
  if (panel) {
    panel.dispose()
    outputChannel.appendLine('[deactivate] Panel disposed')
  }
}