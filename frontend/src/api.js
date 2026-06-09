// API 适配层：自动检测 Wails(桌面) 或 Web(浏览器) 模式
// Web 模式使用 fetch 调用 REST API，Wails 模式使用 Bindings

let _bind = null
let _api = null
let _bindAttempted = false

async function ensureBindings() {
  if (_bindAttempted) return _bind !== null
  _bindAttempted = true

  // 直接检测平台原生桥接（与 @wailsio/runtime 的 _invoke 检测逻辑一致）
  // 在 WebView 中加载时立即可用，不依赖 Go 后端连接状态
  const hasWailsBridge = !!(
    (typeof window.chrome !== 'undefined' && window.chrome.webview?.postMessage) ||
    (typeof window.webkit !== 'undefined' && window.webkit.messageHandlers?.external?.postMessage) ||
    (typeof window.wails !== 'undefined' && window.wails?.invoke)
  )
  if (hasWailsBridge) {
    try {
      _bind = await import('../bindings/ReadBooks/appservice')
      _api = await import('../bindings/ReadBooks/apiservice')
      return true
    } catch {
      // dynamic import 失败 → 浏览器模式
    }
  }
  return false
}

// ---- 通用 fetch 封装 ----
async function apiGet(path) {
  const res = await fetch(path)
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}

async function apiPost(path, body) {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}

async function apiPut(path, body) {
  const res = await fetch(path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}

async function apiDelete(path) {
  const res = await fetch(path, { method: 'DELETE' })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}

// ====== 导出方法 ======

export async function BookList(page, pageSize) {
  if (await ensureBindings()) return _bind.BookList(page, pageSize)
  const data = await apiGet(`/api/books?page=${page}&pageSize=${pageSize}`)
    
  return [data.books || [], data.total || 0]
}

export async function BookGet(idStr) {
  if (await ensureBindings()) return _bind.BookGet(idStr)
  return apiGet(`/api/books?id=${encodeURIComponent(idStr)}`)
}

export async function BookDelete(id) {
  if (await ensureBindings()) return _bind.BookDelete(id)
  return apiDelete(`/api/books?id=${id}`)
}

export async function BookDeleteWithFiles(id) {
  if (await ensureBindings()) return _bind.BookDeleteWithFiles(id)
  return apiDelete(`/api/books?id=${id}&withFiles=true`)
}

export async function BookGetImage(id, page) {
  if (await ensureBindings()) return _bind.BookGetImage(id, page)
  const data = await apiGet(`/api/books/images?id=${id}&page=${page}`)
  return data.images || []
}

export async function BookGetTags(bookID) {
  if (await ensureBindings()) return _bind.BookGetTags(bookID)
  const data = await apiGet(`/api/books/tags?id=${bookID}`)
  return data.tags || []
}

export async function BookSearch(keyword, page, pageSize) {
  if (await ensureBindings()) return _bind.BookSearch(keyword, page, pageSize)
  const data = await apiGet(`/api/books/search?keyword=${encodeURIComponent(keyword)}&page=${page}&pageSize=${pageSize}`)
  return [data.books || [], data.total || 0]
}

export async function BookSetTags(bookID, tagIDs) {
  if (await ensureBindings()) return _bind.BookSetTags(bookID, tagIDs)
  return apiPut(`/api/books/tags?id=${bookID}`, { tag_ids: tagIDs })
}

export async function BookUpdateProgress(bookID, page) {
  if (await ensureBindings()) return _bind.BookUpdateProgress(bookID, page)
  return apiPut(`/api/books/progress?id=${bookID}&page=${page}`)
}

export async function GetChapters(jmid, parent) {
  if (await ensureBindings()) return _bind.GetChapters(jmid, parent)
  const data = await apiGet(`/api/books/chapters?jmid=${jmid}&parent=${parent}`)
  return data.chapters || []
}

export async function AddsComic(path) {
  if (await ensureBindings()) return _bind.AddsComic(path)
  return apiPost('/api/books/import', { path })
}

export async function TagCreate(name, color) {
  if (await ensureBindings()) return _bind.TagCreate(name, color)
  const data = await apiPost('/api/tags', { name, color })
  return data.id
}

export async function TagUpdate(id, name, color) {
  if (await ensureBindings()) return _bind.TagUpdate(id, name, color)
  return apiPut(`/api/tags?id=${id}`, { name, color })
}

export async function TagDelete(id) {
  if (await ensureBindings()) return _bind.TagDelete(id)
  return apiDelete(`/api/tags?id=${id}`)
}

export async function TagListWithCount() {
  if (await ensureBindings()) return _bind.TagListWithCount()
  const data = await apiGet('/api/tags')
  return data.tags || []
}

export async function GetDefaultComicDir() {
  if (await ensureBindings()) return _bind.GetDefaultComicDir()
  const data = await apiGet('/api/settings/comic-dir')
  return data.dir || ''
}

export async function SetDefaultComicDir(dir) {
  if (await ensureBindings()) return _bind.SetDefaultComicDir(dir)
  return apiPut('/api/settings/comic-dir', { dir })
}

export async function SelectFolder() {
  if (await ensureBindings()) return _bind.SelectFolder()
  throw new Error('Windows 文件对话框仅桌面模式可用')
}

// ---- ApiService methods ----
export async function SwitchHttpModel(isOpen) {
  if (await ensureBindings()) return _api.SwitchHttpModel(isOpen)
  return { ok: 'http' }
}

export async function LoadState() {
  if (await ensureBindings()) return _api.LoadState()
  return apiGet('/api/settings')
}

export async function GetHttpLink() {
  if (await ensureBindings()) return _api.GetHttpLink()
  return window.location.origin
}

export async function SaveWindowSize(width, height) {
  if (await ensureBindings()) return _bind.SaveWindowSize(width, height)
}

export async function LoadWindowSize() {
  if (await ensureBindings()) return _bind.LoadWindowSize()
  return [0, 0]
}