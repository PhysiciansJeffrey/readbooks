import { BookList } from '@/api'

const READ_KEY = 'readbooks-random-read-ids'

function loadReadIds() {
  try {
    const raw = localStorage.getItem(READ_KEY)
    return raw ? JSON.parse(raw) : []
  } catch { return [] }
}

function saveReadIds(ids) {
  localStorage.setItem(READ_KEY, JSON.stringify([...ids]))
}

let cachedBooks = null
let readIds = new Set(loadReadIds())

/**
 * 获取一个未读过的随机书籍，持久化已读记录。
 * 全部已读后自动重置。
 */
export async function getUnreadRandomBook() {
  // 无缓存时拉取
  if (!cachedBooks) {
    const [books] = await BookList(1, 10000)
    if (!books || books.length === 0) return null
    cachedBooks = books
  }

  // 过滤未读过
  let available = cachedBooks.filter(b => !readIds.has(b.id))

  // 全部已读 → 重置
  if (available.length === 0) {
    readIds = new Set()
    saveReadIds(readIds)
    const [books] = await BookList(1, 10000)
    if (!books || books.length === 0) return null
    cachedBooks = books
    available = cachedBooks
  }

  const book = available[Math.floor(Math.random() * available.length)]
  readIds.add(book.id)
  saveReadIds(readIds)
  return book
}

/** 清空随机阅读记录和缓存 */
export function clearRandomCache() {
  cachedBooks = null
  readIds = new Set()
  localStorage.removeItem(READ_KEY)
}
