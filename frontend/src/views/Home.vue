<script setup>
import { ref, onMounted, computed } from 'vue'
import { BookList } from '../../bindings/ReadBooks/appservice'
import ComicGrid from '@/components/ComicGrid.vue'
import Pagination from '@/components/Pagination.vue'
import RandomFloatBtn from '@/components/RandomFloatBtn.vue'

const PAGE_SIZE_KEY = 'readbooks-page-size'
const DEFAULT_PAGE_SIZE = 10

const totalBooks = ref(0)
const loading = ref(true)
const error = ref('')
const currentPage = ref(1)
const pageSize = ref(DEFAULT_PAGE_SIZE)
const comics = ref(null)
const totalPages = computed(() => Math.ceil(totalBooks.value / pageSize.value) || 1)

// --- 接口 ---
const fetchComics = async (page = 1) => {
  loading.value = true
  error.value = ''
  currentPage.value = page

  try {
    let [books, total] = await BookList(page, pageSize.value)
    let list = (books || []).filter(Boolean)
    comics.value = list
    totalBooks.value = total
  } catch (e) {
    error.value = e?.message || String(e)
    comics.value = []
    totalBooks.value = 0
  } finally {
    loading.value = false
    // jumpPage.value = page
  }
}

// --- 每页数量 ---
const setPageSize = (e) => {
  let val = pageSize.value
  if (typeof e == 'number') {
    val += e
  } else if (isNaN(val)) {
    val = DEFAULT_PAGE_SIZE
  } else {
    val = parseInt(e.target.value)
  }
  if (val > 50) val = 50
  if (val < 10) val = 10
  if (val === pageSize.value) return
  pageSize.value = val
  localStorage.setItem(PAGE_SIZE_KEY, String(val))
  fetchComics(1)
}

// Pagination 组件的 update:pageSize 事件处理
const onUpdatePageSize = (val) => {
  if (val > 50) val = 50
  if (val < 10) val = 10
  if (val === pageSize.value) return
  pageSize.value = val
  localStorage.setItem(PAGE_SIZE_KEY, String(val))
  fetchComics(1)
}

// --- 翻页 ---
// jumpPage 由 Pagination 组件内部管理

const goToPage = (page) => {
  if (page < 1 || page > totalPages.value || page === currentPage.value) return
  fetchComics(page)
}

const prevPage = () => goToPage(currentPage.value - 1)
const nextPage = () => goToPage(currentPage.value + 1)

onMounted(() => {
  let saved = localStorage.getItem(PAGE_SIZE_KEY)
  if (saved) {
    let val = parseInt(saved, 10)
    if (!isNaN(val) && val >= 5 && val <= 100) {
      pageSize.value = val
    }
  }
  fetchComics(1)
})
</script>

<template>
  <div class="home">
    <!-- 顶部标题栏 -->
    <header class="header">
      <h1 class="header-title">书架列表</h1>
      <div class="header-right">
        <span class="header-count">共 {{ totalBooks }} 本漫画</span>
        <button class="header-refresh" title="刷新" @click="fetchComics(currentPage)" :disabled="loading">
          <span class="refresh-icon" :class="{ spinning: loading }">↻</span>
        </button>
      </div>
    </header>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading">
      <div class="loading-spinner"></div>
      <p>加载中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <button class="retry-btn" @click="fetchComics(currentPage)">重试</button>
    </div>

    <!-- 空状态 -->
    <div v-else-if="comics.length === 0" class="empty">
      <p>书架空空如也</p>
      <button class="import-btn" @click="$router.push('/setting')">去导入</button>
    </div>


    <!-- 漫画网格 + 翻页 -->
    <template v-else>
      <ComicGrid :comics="comics" />
      <Pagination
        :current-page="currentPage"
        :total-pages="totalPages"
        :page-size="pageSize"
        :loading="loading"
        @goto="goToPage"
        @prev="prevPage"
        @next="nextPage"
        @update:pageSize="onUpdatePageSize"
      />
    </template>
  </div>
  <RandomFloatBtn />
</template>
<style scoped>
.home {
  min-height: 100vh;
  padding: 30px 0 80px 0;
  box-sizing: border-box;
}

/* 顶部标题 */
.header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 28px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--sidebar-border);
}

.header-title {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
  color: var(--app-text);
}

.header-count {
  font-size: 14px;
  color: var(--muted-text);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-refresh {
  width: 32px;
  height: 32px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 18px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  outline: none;
  padding: 0;
  transition: border-color 0.12s ease, background-color 0.12s ease;
}

.header-refresh:hover:not(:disabled) {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.header-refresh:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.refresh-icon {
  display: inline-block;
  line-height: 1;
}

.refresh-icon.spinning {
  animation: spin 0.8s linear infinite;
}

/* 加载状态 */
.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  gap: 16px;
  color: var(--muted-text);
}

.loading-spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--sidebar-border);
  border-top-color: var(--app-text);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* 错误状态 */
.error {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80px 0;
  gap: 16px;
  color: #e74c3c;
}

.retry-btn {
  padding: 8px 24px;
  border: 1px solid #e74c3c;
  border-radius: 6px;
  background: transparent;
  color: #e74c3c;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.retry-btn:hover {
  background: #e74c3c;
  color: #fff;
}

/* 空状态 */
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  gap: 20px;
  color: var(--muted-text);
  font-size: 16px;
}

.import-btn {
  padding: 10px 28px;
  border: 1px solid var(--app-text);
  border-radius: 6px;
  background: transparent;
  color: var(--app-text);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.import-btn:hover {
  background: var(--app-text);
  color: var(--app-bg);
}

</style>
