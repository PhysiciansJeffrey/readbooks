<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { BookSearch } from '../../bindings/ReadBooks/appservice'
import ComicGrid from '@/components/ComicGrid.vue'
import Pagination from '@/components/Pagination.vue'
import RandomFloatBtn from '@/components/RandomFloatBtn.vue'

const route = useRoute()

const PAGE_SIZE_KEY = 'readbooks-search-page-size'
const DEFAULT_PAGE_SIZE = 10

const keyword = ref('')
const totalBooks = ref(0)
const loading = ref(true)
const error = ref('')
const currentPage = ref(1)
const pageSize = ref(DEFAULT_PAGE_SIZE)
const comics = ref([])

const totalPages = computed(() => Math.ceil(totalBooks.value / pageSize.value) || 1)

// --- 搜索 ---
const fetchResults = async (page = 1) => {
  loading.value = true
  error.value = ''
  currentPage.value = page

  try {
    const key = keyword.value.trim()
    if (!key) {
      comics.value = []
      totalBooks.value = 0
      loading.value = false
      return
    }
    let [books, total] = await BookSearch(key, page, pageSize.value)
    comics.value = (books || []).filter(Boolean)
    totalBooks.value = total
  } catch (e) {
    error.value = e?.message || String(e)
    comics.value = []
    totalBooks.value = 0
  } finally {
    loading.value = false
  }
}

// --- 每页数量 ---
const onUpdatePageSize = (val) => {
  if (val > 50) val = 50
  if (val < 10) val = 10
  if (val === pageSize.value) return
  pageSize.value = val
  localStorage.setItem(PAGE_SIZE_KEY, String(val))
  fetchResults(1)
}

// --- 翻页 ---
const goToPage = (page) => {
  if (page < 1 || page > totalPages.value || page === currentPage.value) return
  fetchResults(page)
}

const prevPage = () => goToPage(currentPage.value - 1)
const nextPage = () => goToPage(currentPage.value + 1)

// --- 监听路由参数变化 ---
watch(() => route.params.key, (newKey) => {
  keyword.value = decodeURIComponent(newKey || '')
  fetchResults(1)
})

onMounted(() => {
  let saved = localStorage.getItem(PAGE_SIZE_KEY)
  if (saved) {
    let val = parseInt(saved, 10)
    if (!isNaN(val) && val >= 5 && val <= 100) {
      pageSize.value = val
    }
  }
  keyword.value = decodeURIComponent(route.params.key || '')
  fetchResults(1)
})
</script>

<template>
  <div class="search-page">
    <!-- 顶部搜索信息 -->
    <header class="header">
      <h1 class="header-title">
        搜索：<span class="keyword">{{ keyword }}</span>
      </h1>
      <span class="header-count">共 {{ totalBooks }} 条结果</span>
    </header>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading">
      <div class="loading-spinner"></div>
      <p>搜索中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <button class="retry-btn" @click="fetchResults(currentPage)">重试</button>
    </div>

    <!-- 空状态 -->
    <div v-else-if="comics.length === 0" class="empty">
      <p>没有找到相关漫画</p>
    </div>

    <!-- 搜索结果网格 + 翻页 -->
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
.search-page {
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

.keyword {
  color: var(--focus-ring);
}

.header-count {
  font-size: 14px;
  color: var(--muted-text);
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

/* ========== 翻页栏 ========== */
.pagination {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 12px 0;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--sidebar-border);
}
</style>
