const { ipcRenderer } = require("electron");

const MAX_VISIBLE_ITEMS = 3;
const STORAGE_KEY = "devo-recent-folders";

const $ = (sel) => document.querySelector(sel);
const btnOpenFolder = $("#btn-open-folder");
const recentList = $("#recent-list");
const emptyHint = $("#empty-hint");
const btnMore = $("#btn-more");
const modalOverlay = $("#modal-overlay");
const modalBody = $("#modal-body");
const modalList = $("#modal-list");
const modalClose = $("#modal-close");

// ====================== 历史记录管理 ======================
function loadHistory() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveHistory(history) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(history));
}

function addToHistory(folderPath) {
  const history = loadHistory();
  const filtered = history.filter((item) => item.path !== folderPath);
  const folderName = folderPath.split(/[/\\]/).filter(Boolean).pop() || folderPath;
  filtered.unshift({ path: folderPath, name: folderName });
  saveHistory(filtered);
}

function removeFromHistory(folderPath) {
  const history = loadHistory().filter((item) => item.path !== folderPath);
  saveHistory(history);
  return history;
}

// ====================== 渲染 ======================
function renderRecentList() {
  const history = loadHistory();
  const visibleItems = history.slice(0, MAX_VISIBLE_ITEMS);
  const hasMore = history.length > MAX_VISIBLE_ITEMS;

  recentList.innerHTML = "";

  if (history.length === 0) {
    emptyHint.style.display = "block";
    btnMore.style.display = "none";
    return;
  }

  emptyHint.style.display = "none";

  visibleItems.forEach((item) => {
    const li = createHistoryItem(item);
    recentList.appendChild(li);
  });

  btnMore.style.display = hasMore ? "block" : "none";
}

function renderModalList() {
  const history = loadHistory();
  modalList.innerHTML = "";

  history.forEach((item) => {
    const li = createHistoryItem(item);
    modalList.appendChild(li);
  });
}

function createHistoryItem(item) {
  const li = document.createElement("li");
  li.className = "recent-item";
  li.innerHTML = `
    <span class="folder-icon">📁</span>
    <div class="folder-info">
      <div class="folder-name">${escapeHtml(item.name)}</div>
      <div class="folder-path">${escapeHtml(item.path)}</div>
    </div>
    <button class="btn-remove" data-path="${escapeHtml(item.path)}">&times;</button>
  `;

  li.addEventListener("click", (e) => {
    if (e.target.classList.contains("btn-remove")) return;
    openFolder(item.path);
  });

  const removeBtn = li.querySelector(".btn-remove");
  removeBtn.addEventListener("click", (e) => {
    e.stopPropagation();
    li.classList.add("removing");
    li.addEventListener("animationend", () => {
      removeFromHistory(item.path);
      renderRecentList();
    });
  });

  return li;
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

// ====================== 打开文件夹 ======================
function openFolder(folderPath) {
  addToHistory(folderPath);
  ipcRenderer.send("open-recent", folderPath);
}

// ====================== 事件绑定 ======================
btnOpenFolder.addEventListener("click", () => {
  ipcRenderer.send("select-folder");
});

ipcRenderer.on("folder-selected", (_event, folderPath) => {
  if (folderPath) {
    openFolder(folderPath);
  }
});

btnMore.addEventListener("click", () => {
  renderModalList();
  modalOverlay.style.display = "flex";
});

modalClose.addEventListener("click", () => {
  modalOverlay.style.display = "none";
});

modalOverlay.addEventListener("click", (e) => {
  if (e.target === modalOverlay) {
    modalOverlay.style.display = "none";
  }
});

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && modalOverlay.style.display === "flex") {
    modalOverlay.style.display = "none";
  }
});

// ====================== 初始化 ======================
renderRecentList();