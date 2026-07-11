"use strict";var D=Object.create;var m=Object.defineProperty;var I=Object.getOwnPropertyDescriptor;var C=Object.getOwnPropertyNames;var P=Object.getPrototypeOf,B=Object.prototype.hasOwnProperty;var L=(e,o)=>{for(var t in o)m(e,t,{get:o[t],enumerable:!0})},h=(e,o,t,n)=>{if(o&&typeof o=="object"||typeof o=="function")for(let a of C(o))!B.call(e,a)&&a!==t&&m(e,a,{get:()=>o[a],enumerable:!(n=I(o,a))||n.enumerable});return e};var c=(e,o,t)=>(t=e!=null?D(P(e)):{},h(o||!e||!e.__esModule?m(t,"default",{value:e,enumerable:!0}):t,e)),k=e=>h(m({},"__esModule",{value:!0}),e);var H={};L(H,{activate:()=>S,deactivate:()=>j});module.exports=k(H);var r=c(require("vscode")),f=c(require("path")),g=c(require("child_process")),w=c(require("fs")),$=c(require("net")),b=c(require("http")),s,u=[],E=1;function S(e){s=r.window.createOutputChannel("Devo"),e.subscriptions.push(s),s.appendLine("Devo extension activated");let o=r.commands.registerCommand("devo.open",async()=>{await F(e)});e.subscriptions.push(o)}async function F(e){let o=E++,t=r.window.createStatusBarItem(r.StatusBarAlignment.Right,100-u.length);t.command="devo.open",t.text=`$(hubot) Devo # ${o}`,t.tooltip=`Open Devo Instance ${o}`,t.show(),e.subscriptions.push(t);let n={id:o,process:null,port:null,panel:null,statusBarItem:t};u.push(n),s.appendLine(`[instance-${o}] Creating new instance...`);try{await U(n,e)}catch(a){s.appendLine(`[instance-${o}] Failed to start: ${a}`),r.window.showErrorMessage(`Failed to start Devo instance ${o}: ${a}`),v(n);return}T(n,e),y(n)}function T(e,o){let t=e.id;s.appendLine(`[instance-${t}] Creating webview panel on port ${e.port}...`);let n=r.window.createWebviewPanel(`devoPanel-${t}`,`Devo # ${t}`,r.ViewColumn.Beside,{enableScripts:!0,retainContextWhenHidden:!0,localResourceRoots:[]});n.iconPath=r.Uri.joinPath(o.extensionUri,"favicon.svg"),n.onDidDispose(()=>{s.appendLine(`[instance-${t}] Panel disposed`),x(e)}),e.panel=n,n.webview.html=W(e.port,t),s.appendLine(`[instance-${t}] Webview panel created successfully`)}function x(e){let o=e.id;if(s.appendLine(`[instance-${o}] Cleaning up...`),e.process)try{e.process.kill(),s.appendLine(`[instance-${o}] Process killed`)}catch{}e.statusBarItem.dispose(),v(e)}function v(e){let o=u.indexOf(e);o>=0&&u.splice(o,1)}function z(e,o){let t=e.id;e.process=null,e.port=null,s.appendLine(`[instance-${t}] Process crashed with code ${o}`),y(e),e.panel&&(e.panel.webview.html=O(o,t)),r.window.showWarningMessage(`Devo # ${t} process exited (code: ${o??"unknown"}). Click to restart.`,"Restart").then(n=>{n==="Restart"&&(e.panel&&(e.panel.dispose(),e.panel=null),v(e),r.commands.executeCommand("devo.open"))})}function y(e){let o=e.id;e.process&&e.port?(e.statusBarItem.text=`$(pass) Devo # ${o}`,e.statusBarItem.tooltip=`Devo # ${o} running on port ${e.port}`,e.statusBarItem.backgroundColor=void 0):(e.statusBarItem.text=`$(error) Devo # ${o}`,e.statusBarItem.tooltip=`Devo # ${o} stopped`,e.statusBarItem.backgroundColor=new r.ThemeColor("statusBarItem.errorBackground"))}function W(e,o){let t=`http://127.0.0.1:${e}/?mode=vscode`;return`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Devo # ${o}</title>
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
    <span>Connecting to Devo # ${o}...</span>
  </div>
  <iframe id="devo-frame" src="${t}" style="display:none;"
    onload="document.getElementById('loading').style.display='none';this.style.display='block';"
    onerror="document.getElementById('loading').innerHTML='Failed to connect to Devo server at ${t}'">
  </iframe>
</body>
</html>`}function O(e,o){return`<!DOCTYPE html>
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
  <div class="crash-icon">\u{1F4A5}</div>
  <div class="crash-title">Devo # ${o} Crashed</div>
  <div class="crash-message">Exit code: ${e??"unknown"}</div>
</body>
</html>`}function R(){return new Promise((e,o)=>{let t=$.createServer();t.listen(0,"127.0.0.1",()=>{let n=t.address();if(n&&typeof n=="object"){let a=n.port;t.close(()=>e(a))}else t.close(()=>o(new Error("Failed to get port")))}),t.on("error",o)})}function M(e){let t=process.platform==="win32"?"devo.exe":"devo",n=f.join(e.extensionPath,"bin",t);return w.existsSync(n)?n:"devo"}function N(e,o){return new Promise((t,n)=>{let a=`http://127.0.0.1:${e}/api/v1/health`,l=Date.now();function p(){if(Date.now()-l>o)return n(new Error("Devo startup timed out after 30 seconds"));let d=b.get(a,i=>{i.statusCode===200?(i.resume(),t()):(i.resume(),setTimeout(p,500))});d.on("error",()=>{setTimeout(p,500)}),d.setTimeout(3e3,()=>{d.destroy(),setTimeout(p,500)})}p()})}async function U(e,o){let t=e.id,n=r.workspace.workspaceFolders?.[0]?.uri.fsPath;e.port=await R(),s.appendLine(`[instance-${t}] Found free port: ${e.port}`);let a=["--port",String(e.port)];n&&(a.push("--workspace",n),s.appendLine(`[instance-${t}] Workspace: ${n}`));let l=M(o);return s.appendLine(`[instance-${t}] Binary path: ${l}`),s.appendLine(`[instance-${t}] Spawning: ${l} ${a.join(" ")}`),new Promise((p,d)=>{e.process=g.spawn(l,a,{stdio:["pipe","pipe","pipe"],env:{...process.env}}),s.appendLine(`[instance-${t}] PID: ${e.process.pid}`),e.process.stdout?.on("data",i=>{s.appendLine(`[instance-${t}][stdout] ${i.toString().trim()}`)}),e.process.stderr?.on("data",i=>{s.appendLine(`[instance-${t}][stderr] ${i.toString().trim()}`)}),e.process.on("error",i=>{e.process=null,e.port=null,s.appendLine(`[instance-${t}] Error: ${i.message}`),d(new Error(`Cannot start devo: ${i.message}.`))}),e.process.on("close",i=>{s.appendLine(`[instance-${t}] Process exited with code ${i}`),e.port===null?d(new Error(`Devo exited with code ${i} before port was assigned`)):z(e,i)}),N(e.port,3e4).then(()=>{s.appendLine(`[instance-${t}] Server started successfully`),p()}).catch(i=>{s.appendLine(`[instance-${t}] Startup timed out after 30 seconds`),d(i)})})}function j(){s?.appendLine("[deactivate] Deactivating extension, cleaning up all instances...");for(let e of[...u])x(e)}0&&(module.exports={activate,deactivate});
