# Devo - AI Coding Agent

**一个会自己写代码、自己跑、报错了自己修、修不好再来、直到搞定的 AI 编码助手。**

你说话，它干活。你只需要在它要动手的时候点个头。

---

## 如何打开

| 方式 | 位置 |
|------|------|
| 编辑器工具栏 | 编辑器右上角的 Devo 图标 |
| 命令面板 | `Ctrl+Shift+P` → 输入 `Open Devo` |

---

## 典型工作流

1. 打开 Devo，创建会话，指定你的项目目录
2. 说需求：**"帮我写一个下载网页图片的脚本"**
3. AI 开始分析，可能先搜索项目里有什么，然后生成代码
4. 弹窗让你审代码，你批准，它写入文件
5. AI 自己跑脚本，报错了，它分析错误，再改，再跑
6. 反复几次，跑通了，告诉你搞定
7. 不满意中间某步？回滚聊天记录，用 Git 回退代码，重新来

---

## 项目规则（agents.md）

在项目根目录放一个 `agents.md`，Devo 就会按照你的规则来干活：

```markdown
# agents.md

## 编码规范
- 所有函数必须写 JSDoc 注释
- 使用 TypeScript 严格模式
- 组件放在 src/components/ 下

## 项目约定
- 环境变量从 .env.local 读取
- 不要修改 package.json 的依赖版本
```

你可以写任何东西——编码风格、命名规范、目录结构、禁止事项、常用命令。Devo 每次对话都会先读这个文件，确保不越界。

---

## 安装

### 从 VSIX 安装

1. 下载最新的 `.vsix` 文件
2. VS Code 中打开 **Extensions**（`Ctrl+Shift+X`）
3. 点击右上角 `...` → **Install from VSIX...**
4. 选择 `.vsix` 文件，安装完成

### 从源码构建

```bash
cd vscode-extension
npm install
npm run compile
npm run vsix
```

---

## 调试

打开 **Output** 面板（`View` → `Output`），右侧下拉选择 **Devo**，可查看扩展日志。

## 要求

- VS Code `^1.85.0`
- Devo 核心程序（随扩展一起分发在 `bin/` 目录中）