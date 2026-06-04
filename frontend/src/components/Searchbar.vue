<script setup>
import { ref, onBeforeUnmount, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const keyword = ref('')
const isHovered = ref(false)
const isFocused = ref(false)
const isPinned = ref(false)
const show = ref(false)

const HISTORY_KEY = 'readbooks-search-history'
const HISTORY_MAX = 10

const searchHistory = ref([])

const loadHistory = () => {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    if (raw) searchHistory.value = JSON.parse(raw)
  } catch { searchHistory.value = [] }
}

const saveHistory = (kw) => {
  if (!kw) return
  const list = searchHistory.value.filter(item => item !== kw)
  list.unshift(kw)
  if (list.length > HISTORY_MAX) list.length = HISTORY_MAX
  searchHistory.value = list
  localStorage.setItem(HISTORY_KEY, JSON.stringify(list))
}

const removeHistory = (e, kw) => {
  e.stopPropagation()
  searchHistory.value = searchHistory.value.filter(item => item !== kw)
  localStorage.setItem(HISTORY_KEY, JSON.stringify(searchHistory.value))
}

const clearHistory = () => {
  searchHistory.value = []
  localStorage.removeItem(HISTORY_KEY)
}

const onSearch = (kw) => {
  const q = (kw || keyword.value).trim()
  if (!q) return
  saveHistory(q)
  keyword.value = ''
  isPinned.value = false
  isFocused.value = false
  show.value = false
  router.push(`/search/${encodeURIComponent(q)}`)
}

let showTimeout = null
let hideTimeout = null

const onTopHover = () => {
  clearTimeout(hideTimeout)
  showTimeout = setTimeout(() => {
    isHovered.value = true
    show.value = true
  }, 80)
}

const onTopLeave = () => {
  clearTimeout(showTimeout)
  hideTimeout = setTimeout(() => {
    isHovered.value = false
    if (!isPinned.value) show.value = false
  }, 150)
}

defineExpose({ onTopHover, onTopLeave })

const onMouseEnter = () => {
  clearTimeout(hideTimeout)
  show.value = true
}

const onMouseLeave = () => {
  if (!isPinned.value && !isFocused.value) {
    hideTimeout = setTimeout(() => {
      if (!isHovered.value) show.value = false
    }, 200)
  }
}

const onFocus = () => {
  isFocused.value = true
  isPinned.value = true
  show.value = true
}

const onBlur = () => {
  isFocused.value = false
  if (!isPinned.value) {
    hideTimeout = setTimeout(() => {
      if (!isHovered.value && !isPinned.value) show.value = false
    }, 200)
  }
}

const onOverlayClick = () => {
  isPinned.value = false
  isFocused.value = false
  keyword.value = ''
  show.value = false
}

onMounted(loadHistory)

onBeforeUnmount(() => {
  clearTimeout(showTimeout)
  clearTimeout(hideTimeout)
})
</script>

<template>
  <!-- 灰色蒙版 -->
  <Transition name="overlay-fade">
    <div v-if="isPinned" class="search-overlay" @click="onOverlayClick"></div>
  </Transition>

  <!-- 搜索栏 -->
  <Transition name="search-fade">
    <div v-if="show" class="searchbar" :class="{ expanded: isPinned }" @mouseenter="onMouseEnter"
      @mouseleave="onMouseLeave">
      <div style="display: flex;">
        <!-- 搜索框 -->
        <input v-model="keyword" type="text" class="search-input" placeholder="搜索漫画..." aria-label="搜索关键词"
          @focus="onFocus" @blur="onBlur" @keyup.enter="onSearch()" />
        <span class="search-sep">|</span>
        <!-- 搜索按钮 -->
        <button class="search-btn" type="button" aria-label="搜索" @click="onSearch()">
          <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
        </button>
      </div>
      <!-- 历史搜索标签 -->
      <div v-if="isPinned && searchHistory.length > 0" class="history-tags">
        <button v-for="item in searchHistory" :key="item" class="history-tag" @click="onSearch(item)">
          <span class="history-tag-text">{{ item }}</span>
          <span class="history-tag-del" @click="removeHistory($event, item)">×</span>
        </button>
        <button class="history-clear" @click="clearHistory">清空</button>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
/* 灰色蒙版 */
.search-overlay {
  position: fixed;
  inset: 0;
  z-index: 19;
  background: rgba(0, 0, 0, 0.35);
}

.overlay-fade-enter-active,
.overlay-fade-leave-active {
  transition: opacity 0.15s ease;
}

.overlay-fade-enter-from,
.overlay-fade-leave-to {
  opacity: 0;
}

/* 搜索栏容器 */
.searchbar {
  position: fixed;
  top: 18px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 20;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  border: 1px solid var(--sidebar-border);
  border-radius: 12px;
  background: var(--sidebar-bg);
  overflow: hidden;
  min-width: 280px;
  width: 40%;
  max-width: 600px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}

.searchbar.expanded {
  border-color: var(--app-text);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
}

/* 搜索栏淡入淡出 */
.search-fade-enter-active,
.search-fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.search-fade-enter-from,
.search-fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-8px);
}

/* 搜索行 */
.search-input {
  flex: 1;
  min-width: 0;
  height: 42px;
  padding: 0 20px;
  border: none;
  background: transparent;
  color: var(--app-text);
  font-size: 14px;
  outline: none;
}

.search-input::placeholder {
  color: var(--muted-text);
}

.search-sep {
  display: flex;
  align-items: center;
  padding: 0 4px;
  color: var(--sidebar-border);
  font-size: 14px;
  user-select: none;
}

.search-btn {
  border: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 42px;
  height: 42px;
  padding: 0;
  background: transparent;
  color: var(--muted-text);
  cursor: pointer;
  transition: color 0.12s ease;
}

.search-btn:hover {
  color: var(--app-text);
}

.search-btn:focus-visible {
  outline: 2px solid var(--focus-ring);
  outline-offset: -2px;
  border-radius: 50%;
}

.search-icon {
  width: 18px;
  height: 18px;
}

/* 历史搜索标签 */
.history-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 10px 14px 12px;
  border-top: 1px solid var(--sidebar-border);
}

.history-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--switch-border);
  border-radius: 999px;
  background: transparent;
  color: var(--app-text);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.12s ease, border-color 0.12s ease;
}

.history-tag:hover {
  background: var(--switch-bg);
  border-color: var(--app-text);
}

.history-tag-text {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.history-tag-del {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  font-size: 14px;
  line-height: 1;
  color: var(--muted-text);
  cursor: pointer;
  flex-shrink: 0;
}

.history-tag-del:hover {
  background: var(--switch-border);
  color: var(--app-text);
}

.history-clear {
  padding: 4px 10px;
  border: 1px dashed var(--switch-border);
  border-radius: 999px;
  background: transparent;
  color: var(--muted-text);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.12s ease;
}

.history-clear:hover {
  border-color: var(--app-text);
  color: var(--app-text);
}
</style>
