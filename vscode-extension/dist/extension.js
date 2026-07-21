"use strict";var x=Object.create;var u=Object.defineProperty;var C=Object.getOwnPropertyDescriptor;var D=Object.getOwnPropertyNames;var P=Object.getPrototypeOf,L=Object.prototype.hasOwnProperty;var I=(e,t)=>{for(var n in t)u(e,n,{get:t[n],enumerable:!0})},v=(e,t,n,o)=>{if(t&&typeof t=="object"||typeof t=="function")for(let a of D(t))!L.call(e,a)&&a!==n&&u(e,a,{get:()=>t[a],enumerable:!(o=C(t,a))||o.enumerable});return e};var p=(e,t,n)=>(n=e!=null?x(P(e)):{},v(t||!e||!e.__esModule?u(n,"default",{value:e,enumerable:!0}):n,e)),k=e=>v(u({},"__esModule",{value:!0}),e);var j={};I(j,{activate:()=>F,deactivate:()=>U});module.exports=k(j);var r=p(require("vscode")),f=p(require("path")),g=p(require("child_process")),w=p(require("fs")),b=p(require("net")),$=p(require("http")),s,m=[],E=1;function F(e){s=r.window.createOutputChannel("Devo"),e.subscriptions.push(s),s.appendLine("Devo extension activated");let t=r.commands.registerCommand("devo.open",async()=>{await S(e)});e.subscriptions.push(t)}async function S(e){let t=E++,n={id:t,process:null,port:null,panel:null};m.push(n),s.appendLine(`[instance-${t}] Creating new instance...`);try{await N(n,e)}catch(o){s.appendLine(`[instance-${t}] Failed to start: ${o}`),r.window.showErrorMessage(`Failed to start Devo instance ${t}: ${o}`),h(n);return}T(n,e)}function T(e,t){let n=e.id;s.appendLine(`[instance-${n}] Creating webview panel on port ${e.port}...`);let o=r.window.createWebviewPanel(`devoPanel-${n}`,`Devo # ${n}`,r.ViewColumn.Beside,{enableScripts:!0,retainContextWhenHidden:!0,localResourceRoots:[]});o.iconPath=r.Uri.joinPath(t.extensionUri,"favicon.svg"),o.onDidDispose(()=>{s.appendLine(`[instance-${n}] Panel disposed`),y(e)}),e.panel=o,o.webview.html=B(e.port,n),s.appendLine(`[instance-${n}] Webview panel created successfully`)}function y(e){let t=e.id;if(s.appendLine(`[instance-${t}] Cleaning up...`),e.process)try{e.process.kill(),s.appendLine(`[instance-${t}] Process killed`)}catch{}h(e)}function h(e){let t=m.indexOf(e);t>=0&&m.splice(t,1)}function z(e,t){let n=e.id;e.process=null,e.port=null,s.appendLine(`[instance-${n}] Process crashed with code ${t}`),e.panel&&(e.panel.webview.html=W(t,n)),r.window.showWarningMessage(`Devo # ${n} process exited (code: ${t??"unknown"}). Click to restart.`,"Restart").then(o=>{o==="Restart"&&(e.panel&&(e.panel.dispose(),e.panel=null),h(e),r.commands.executeCommand("devo.open"))})}function B(e,t){let n=`http://127.0.0.1:${e}/?mode=vscode`;return`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Devo # ${t}</title>
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
    <span>Connecting to Devo # ${t}...</span>
  </div>
  <iframe id="devo-frame" src="${n}" style="display:none;"
    onload="document.getElementById('loading').style.display='none';this.style.display='block';"
    onerror="document.getElementById('loading').innerHTML='Failed to connect to Devo server at ${n}'">
  </iframe>
</body>
</html>`}function W(e,t){return`<!DOCTYPE html>
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
  <div class="crash-title">Devo # ${t} Crashed</div>
  <div class="crash-message">Exit code: ${e??"unknown"}</div>
</body>
</html>`}function M(){return new Promise((e,t)=>{let n=b.createServer();n.listen(0,"127.0.0.1",()=>{let o=n.address();if(o&&typeof o=="object"){let a=o.port;n.close(()=>e(a))}else n.close(()=>t(new Error("Failed to get port")))}),n.on("error",t)})}function O(e){let n=process.platform==="win32"?"devo.exe":"devo",o=f.join(e.extensionPath,"bin",n);return w.existsSync(o)?o:"devo"}function R(e,t){return new Promise((n,o)=>{let a=`http://127.0.0.1:${e}/api/v1/health`,l=Date.now();function c(){if(Date.now()-l>t)return o(new Error("Devo startup timed out after 30 seconds"));let d=$.get(a,i=>{i.statusCode===200?(i.resume(),n()):(i.resume(),setTimeout(c,500))});d.on("error",()=>{setTimeout(c,500)}),d.setTimeout(3e3,()=>{d.destroy(),setTimeout(c,500)})}c()})}async function N(e,t){let n=e.id,o=r.workspace.workspaceFolders?.[0]?.uri.fsPath;e.port=await M(),s.appendLine(`[instance-${n}] Found free port: ${e.port}`);let a=["--port",String(e.port)];o&&(a.push("--workspace",o),s.appendLine(`[instance-${n}] Workspace: ${o}`));let l=O(t);return s.appendLine(`[instance-${n}] Binary path: ${l}`),s.appendLine(`[instance-${n}] Spawning: ${l} ${a.join(" ")}`),new Promise((c,d)=>{e.process=g.spawn(l,a,{stdio:["pipe","pipe","pipe"],env:{...process.env}}),s.appendLine(`[instance-${n}] PID: ${e.process.pid}`),e.process.stdout?.on("data",i=>{s.appendLine(`[instance-${n}][stdout] ${i.toString().trim()}`)}),e.process.stderr?.on("data",i=>{s.appendLine(`[instance-${n}][stderr] ${i.toString().trim()}`)}),e.process.on("error",i=>{e.process=null,e.port=null,s.appendLine(`[instance-${n}] Error: ${i.message}`),d(new Error(`Cannot start devo: ${i.message}.`))}),e.process.on("close",i=>{s.appendLine(`[instance-${n}] Process exited with code ${i}`),e.port===null?d(new Error(`Devo exited with code ${i} before port was assigned`)):z(e,i)}),R(e.port,3e4).then(()=>{s.appendLine(`[instance-${n}] Server started successfully`),c()}).catch(i=>{s.appendLine(`[instance-${n}] Startup timed out after 30 seconds`),d(i)})})}function U(){s?.appendLine("[deactivate] Deactivating extension, cleaning up all instances...");for(let e of[...m])y(e)}0&&(module.exports={activate,deactivate});
ndLine(`[instance-${t}] Error: ${i.message}`),d(new Error(`Cannot start devo: ${i.message}.`))}),e.process.on("close",i=>{s.appendLine(`[instance-${t}] Process exited with code ${i}`),e.port===null?d(new Error(`Devo exited with code ${i} before port was assigned`)):z(e,i)}),N(e.port,3e4).then(()=>{s.appendLine(`[instance-${t}] Server started successfully`),p()}).catch(i=>{s.appendLine(`[instance-${t}] Startup timed out after 30 seconds`),d(i)})})}function j(){s?.appendLine("[deactivate] Deactivating extension, cleaning up all instances...");for(let e of[...u])x(e)}0&&(module.exports={activate,deactivate});
