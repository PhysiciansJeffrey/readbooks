<script setup>
import { computed } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { BookList } from '@/api'

const props = defineProps({
  pages: {
    type: Array,
    default: () => [],
  },
})

const route = useRoute()
const router = useRouter()

// 缓存 bookList，避免每次点击都请求
let cachedBooks = null

// 已随机过的 book id（localStorage 持久化）
const READ_KEY = 'readbooks-random-read-ids'
let readIds = new Set(loadReadIds())

function loadReadIds() {
  try {
    const raw = localStorage.getItem(READ_KEY)
    return raw ? JSON.parse(raw) : []
  } catch { return [] }
}

function saveReadIds() {
  localStorage.setItem(READ_KEY, JSON.stringify([...readIds]))
}

const visible = computed(() => {
  if (!props.pages || props.pages.length === 0) return true
  return props.pages.some(p => {
    if (p.endsWith('*')) {
      return route.name?.toString().startsWith(p.slice(0, -1))
    }
    return route.name === p
  })
})

function clearCache() {
  cachedBooks = null
  readIds = new Set()
  localStorage.removeItem(READ_KEY)
}

const randomRead = async () => {
  try {
    // 无缓存时拉取
    if (!cachedBooks) {
      const [books] = await BookList(1, 10000)
      if (!books || books.length === 0) return
      cachedBooks = books
    }

    // 过滤未读过
    let available = cachedBooks.filter(b => !readIds.has(b.id))

    // 全部已读 → 重置缓存和记录
    if (available.length === 0) {
      readIds = new Set()
      saveReadIds()
      const [books] = await BookList(1, 10000)
      if (!books || books.length === 0) return
      cachedBooks = books
      available = cachedBooks
    }

    const randomBook = available[Math.floor(Math.random() * available.length)]
    readIds.add(randomBook.id)
    saveReadIds()
    router.push(`/resume/${randomBook.id}`)
  } catch (e) {
    console.error('随机跳转失败:', e)
  }
}
</script>

<template>
  <div v-if="visible" class="float-btn-group">
    <button class="random-float-btn" title="随机观看" @click="randomRead">?</button>
    <button class="clear-cache-btn" title="重置随机记录" @click="clearCache">↺</button>
  </div>
</template>

<style scoped>
.float-btn-group {
  position: fixed;
  bottom: 70px;
  right: 5px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  z-index: 11;
}

.random-float-btn {
  width: 44px;
  height: 44px;
  border: 1px solid var(--switch-border);
  border-radius: 8px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 18px;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  outline: none;
  transition: background-color 0.12s ease, border-color 0.12s ease;
}

.random-float-btn:hover {
  background: var(--switch-bg);
  border-color: var(--app-text);
}

.random-float-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.clear-cache-btn {
  width: 28px;
  height: 28px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0.5;
  outline: none;
  transition: opacity 0.12s ease;
}

.clear-cache-btn:hover {
  opacity: 1;
}
</style>
