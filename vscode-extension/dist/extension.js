"use strict";var D=Object.create;var u=Object.defineProperty;var C=Object.getOwnPropertyDescriptor;var P=Object.getOwnPropertyNames;var B=Object.getPrototypeOf,L=Object.prototype.hasOwnProperty;var k=(e,n)=>{for(var t in n)u(e,t,{get:n[t],enumerable:!0})},g=(e,n,t,o)=>{if(n&&typeof n=="object"||typeof n=="function")for(let a of P(n))!L.call(e,a)&&a!==t&&u(e,a,{get:()=>n[a],enumerable:!(o=C(n,a))||o.enumerable});return e};var c=(e,n,t)=>(t=e!=null?D(B(e)):{},g(n||!e||!e.__esModule?u(t,"default",{value:e,enumerable:!0}):t,e)),S=e=>g(u({},"__esModule",{value:!0}),e);var H={};k(H,{activate:()=>F,deactivate:()=>j});module.exports=S(H);var r=c(require("vscode")),w=c(require("path")),$=c(require("child_process")),b=c(require("fs")),x=c(require("net")),s,p=[],E=1;function F(e){s=r.window.createOutputChannel("Devo"),e.subscriptions.push(s),s.appendLine("Devo extension activated");let n=r.commands.registerCommand("devo.open",async()=>{await T(e)});e.subscriptions.push(n)}async function T(e){let n=E++,t=r.window.createStatusBarItem(r.StatusBarAlignment.Right,100-p.length);t.command="devo.open",t.text=`$(hubot) Devo # ${n}`,t.tooltip=`Open Devo Instance ${n}`,t.show(),e.subscriptions.push(t);let o={id:n,process:null,port:null,panel:null,statusBarItem:t};p.push(o),s.appendLine(`[instance-${n}] Creating new instance...`);try{await U(o,e)}catch(a){s.appendLine(`[instance-${n}] Failed to start: ${a}`),r.window.showErrorMessage(`Failed to start Devo instance ${n}: ${a}`),h(o);return}z(o,e),I(o)}function z(e,n){let t=e.id;s.appendLine(`[instance-${t}] Creating webview panel on port ${e.port}...`);let o=r.window.createWebviewPanel(`devoPanel-${t}`,`Devo # ${t}`,r.ViewColumn.Beside,{enableScripts:!0,retainContextWhenHidden:!0,localResourceRoots:[]});o.iconPath=r.Uri.joinPath(n.extensionUri,"favicon.svg"),o.onDidDispose(()=>{s.appendLine(`[instance-${t}] Panel disposed`),y(e)}),e.panel=o,o.webview.html=O(e.port,t),s.appendLine(`[instance-${t}] Webview panel created successfully`)}function y(e){let n=e.id;if(s.appendLine(`[instance-${n}] Cleaning up...`),e.process)try{e.process.kill(),s.appendLine(`[instance-${n}] Process killed`)}catch{}e.statusBarItem.dispose(),h(e)}function h(e){let n=p.indexOf(e);n>=0&&p.splice(n,1)}function W(e,n){let t=e.id;e.process=null,e.port=null,s.appendLine(`[instance-${t}] Process crashed with code ${n}`),I(e),e.panel&&(e.panel.webview.html=R(n,t)),r.window.showWarningMessage(`Devo # ${t} process exited (code: ${n??"unknown"}). Click to restart.`,"Restart").then(o=>{o==="Restart"&&(e.panel&&(e.panel.dispose(),e.panel=null),h(e),r.commands.executeCommand("devo.open"))})}function I(e){let n=e.id;e.process&&e.port?(e.statusBarItem.text=`$(pass) Devo # ${n}`,e.statusBarItem.tooltip=`Devo # ${n} running on port ${e.port}`,e.statusBarItem.backgroundColor=void 0):(e.statusBarItem.text=`$(error) Devo # ${n}`,e.statusBarItem.tooltip=`Devo # ${n} stopped`,e.statusBarItem.backgroundColor=new r.ThemeColor("statusBarItem.errorBackground"))}function O(e,n){let t=`http://127.0.0.1:${e}/?mode=vscode`;return`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Devo # ${n}</title>
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
    <span>Connecting to Devo # ${n}...</span>
  </div>
  <iframe id="devo-frame" src="${t}" style="display:none;"
    onload="document.getElementById('loading').style.display='none';this.style.display='block';"
    onerror="document.getElementById('loading').innerHTML='Failed to connect to Devo server at ${t}'">
  </iframe>
</body>
</html>`}function R(e,n){return`<!DOCTYPE html>
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
  <div class="crash-title">Devo # ${n} Crashed</div>
  <div class="crash-message">Exit code: ${e??"unknown"}</div>
</body>
</html>`}function M(){return new Promise((e,n)=>{let t=x.createServer();t.listen(0,"127.0.0.1",()=>{let o=t.address();if(o&&typeof o=="object"){let a=o.port;t.close(()=>e(a))}else t.close(()=>n(new Error("Failed to get port")))}),t.on("error",n)})}function N(e){let t=process.platform==="win32"?"devo.exe":"devo",o=w.join(e.extensionPath,"bin",t);return b.existsSync(o)?o:"devo"}async function U(e,n){let t=e.id,o=r.workspace.workspaceFolders?.[0]?.uri.fsPath;e.port=await M(),s.appendLine(`[instance-${t}] Found free port: ${e.port}`);let a=["--port",String(e.port)];o&&(a.push("--workspace",o),s.appendLine(`[instance-${t}] Workspace: ${o}`));let m=N(n);return s.appendLine(`[instance-${t}] Binary path: ${m}`),s.appendLine(`[instance-${t}] Spawning: ${m} ${a.join(" ")}`),new Promise((f,v)=>{e.process=$.spawn(m,a,{stdio:["pipe","pipe","pipe"],env:{...process.env}}),s.appendLine(`[instance-${t}] PID: ${e.process.pid}`);let l=setTimeout(()=>{s.appendLine(`[instance-${t}] Startup timed out after 30 seconds`),v(new Error("Devo startup timed out after 30 seconds"))},3e4);e.process.stdout?.on("data",i=>{let d=i.toString();s.appendLine(`[instance-${t}][stdout] ${d.trim()}`),d.includes("Server ready")&&(clearTimeout(l),s.appendLine(`[instance-${t}] Server started successfully`),f())}),e.process.stderr?.on("data",i=>{let d=i.toString();s.appendLine(`[instance-${t}][stderr] ${d.trim()}`),d.includes("Server ready")&&(clearTimeout(l),s.appendLine(`[instance-${t}] Server started successfully (via stderr)`),f())}),e.process.on("error",i=>{clearTimeout(l),e.process=null,e.port=null,s.appendLine(`[instance-${t}] Error: ${i.message}`),v(new Error(`Cannot start devo: ${i.message}.`))}),e.process.on("close",i=>{clearTimeout(l),s.appendLine(`[instance-${t}] Process exited with code ${i}`),e.port===null?v(new Error(`Devo exited with code ${i} before port was assigned`)):W(e,i)})})}function j(){s?.appendLine("[deactivate] Deactivating extension, cleaning up all instances...");for(let e of[...p])y(e)}0&&(module.exports={activate,deactivate});
