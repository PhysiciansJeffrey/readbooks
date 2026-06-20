<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BookGet, BookGetImage, BookUpdateProgress, GetChapters, BookList } from '@/api'

const route = useRoute()
const router = useRouter()

// ========== 状态 ==========
const loading = ref(true)
const error = ref('')
const book = ref(null)
const images = ref([])
const totalPages = ref(0)
const chapters = ref([])

// 是否有上一章/下一章
const hasPrevChapter = computed(() => {
  if (!book.value || chapters.value.length === 0) return false
  const idx = chapters.value.findIndex(c => c.id === book.value.id)
  return idx > 0
})
const hasNextChapter = computed(() => {
  if (!book.value || chapters.value.length === 0) return false
  const idx = chapters.value.findIndex(c => c.id === book.value.id)
  return idx >= 0 && idx < chapters.value.length - 1
})

const prevChapter = () => {
  const idx = chapters.value.findIndex(c => c.id === book.value?.id)
  if (idx <= 0) return
  stopAutoPlay()
  router.push(`/detail/${chapters.value[idx - 1].id}`)
}

const nextChapter = () => {
  const idx = chapters.value.findIndex(c => c.id === book.value?.id)
  if (idx < 0 || idx >= chapters.value.length - 1) return
  stopAutoPlay()
  router.push(`/detail/${chapters.value[idx + 1].id}`)
}

const viewMode = ref('horizontal')
const displayPage = ref(1)   // 当前页码（从1开始，统一用实际图片页码）
const preloadCache = new Map()
const showPageSelect = ref(false)
const PROGRESS_KEY_PREFIX = 'readbooks-progress-'
const VIEW_MODE_KEY = 'readbooks-view-mode'
const AUTOPLAY_KEY = 'readbooks-autoplay'

// ========== 自动播放 ==========
const autoPlaying = ref(false)
const countdown = ref(null)
let autoPlayTimer = null
let countdownTimer = null

const goToRandomBook = async () => {
  try {
    const [books] = await BookList(1, 10000)
    if (!books || books.length === 0) return
    const book = books[Math.floor(Math.random() * books.length)]
    router.push(`/detail/${book.id}`)
  } catch (e) {
    console.error('随机跳转失败:', e)
  }
}

const cancelCountdown = () => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  countdown.value = null
}

const startCountdown = () => {
  countdown.value = 5
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      cancelCountdown()
      goToRandomBook()
    }
  }, 1000)
}

const resetCountdown = () => {
  if (countdown.value === null) return
  if (countdownTimer) clearInterval(countdownTimer)
  countdown.value = 5
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      cancelCountdown()
      goToRandomBook()
    }
  }, 1000)
}

const tryNextChapter = async () => {
  if (!book.value || book.value.parent < 1) {
    startCountdown()
    return
  }
  try {
    const chapters = await GetChapters(book.value.jmid, book.value.parent)
    if (!chapters || chapters.length === 0) {
      startCountdown()
      return
    }
    const currentId = book.value.id
    const idx = chapters.findIndex(c => c.id === currentId)
    if (idx < 0 || idx >= chapters.length - 1) {
      startCountdown()
      return
    }
    const nextBook = chapters[idx + 1]
    router.push(`/detail/${nextBook.id}`)
  } catch (e) {
    console.error('获取下一章失败:', e)
    startCountdown()
  }
}

const startAutoPlay = () => {
  if (autoPlaying.value) return
  autoPlaying.value = true
  autoPlayTimer = setInterval(() => {
    if (displayPage.value >= totalPages.value) {
      clearInterval(autoPlayTimer)
      autoPlayTimer = null
      autoPlaying.value = false
      tryNextChapter()
      return
    }
    displayPage.value++
    preloadImages(normalizedPage.value - 1)
    saveProgress()
    scrollToCurrentPage()
  }, 4000)
}

const stopAutoPlay = () => {
  cancelCountdown()
  autoPlaying.value = false
  if (autoPlayTimer) {
    clearInterval(autoPlayTimer)
    autoPlayTimer = null
  }
}

const toggleAutoPlay = () => {
  if (countdown.value !== null) {
    cancelCountdown()
    localStorage.setItem(AUTOPLAY_KEY, '0')
    return
  }
  if (autoPlaying.value) {
    stopAutoPlay()
    localStorage.setItem(AUTOPLAY_KEY, '0')
  } else {
    startAutoPlay()
    localStorage.setItem(AUTOPLAY_KEY, '1')
  }
}

// ========== 竖屏拖拽滚动 ==========
const isDragging = ref(false)
const dragStartY = ref(0)
const dragStartScrollY = ref(0)

const onVerticalMousedown = (e) => {
  if (viewMode.value !== 'vertical') return
  isDragging.value = true
  dragStartY.value = e.clientY
  dragStartScrollY.value = window.scrollY
  document.body.style.cursor = 'grabbing'
  document.body.style.userSelect = 'none'
  e.preventDefault()
}

const onVerticalMousemove = (e) => {
  if (!isDragging.value) return
  const dy = e.clientY - dragStartY.value
  const newY = dragStartScrollY.value - dy
  window.scrollTo({ top: newY, behavior: 'instant' })
}

const onVerticalMouseup = () => {
  if (!isDragging.value) return
  isDragging.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}



// ========== 计算属性 ==========

// 横屏：起始页必须是奇数（1,3,5...），用于 currentPages 计算
const normalizedPage = computed(() => {
  if (viewMode.value !== 'horizontal') return displayPage.value
  // 横屏模式下，页码对齐到奇数：显示 (p, p+1)
  const p = displayPage.value
  return p % 2 === 0 ? p - 1 : p
})

const currentPages = computed(() => {
  if (viewMode.value !== 'horizontal') return []
  const idx = normalizedPage.value - 1
  return [
    images.value[idx] || null,
    images.value[idx + 1] || null,
  ]
})

const pageList = computed(() => {
  const pages = []
  for (let i = 1; i <= totalPages.value; i++) pages.push(i)
  return pages
})

// 浮动按钮页码文字
const pageText = computed(() => {
  if (viewMode.value === 'horizontal') {
    const end = Math.min(normalizedPage.value + 1, totalPages.value)
    return normalizedPage.value === end
      ? `${normalizedPage.value}/${totalPages.value}`
      : `${normalizedPage.value}-${end}/${totalPages.value}`
  }
  return `${displayPage.value}/${totalPages.value}`
})

const imageUrl = (path) => {
  if (!path) return ''
  return `/api/image?p=${encodeURIComponent(path)}`
}

// ========== 预加载 ==========
const preloadImages = (startIdx) => {
  for (let i = startIdx; i < startIdx + 4 && i < images.value.length; i++) {
    if (preloadCache.has(i)) continue
    const img = new Image()
    img.src = imageUrl(images.value[i])
    preloadCache.set(i, img)
  }
}

// ========== 保存进度 ==========
const saveProgress = () => {
  if (!book.value) return
  localStorage.setItem(`${PROGRESS_KEY_PREFIX}${book.value.id}`, String(displayPage.value))
}

// ========== 翻页 ==========
const scrollToCurrentPage = () => {
  if (viewMode.value !== 'vertical') return
  setTimeout(() => {
    const target = document.querySelector(`.v-page:nth-child(${displayPage.value})`)
    if (target) {
      const rect = target.getBoundingClientRect()
      const scrollY = window.scrollY + rect.top
      window.scrollTo({ top: scrollY, behavior: 'instant' })
    }
  }, 50)
}


const prevPage = () => {
  resetCountdown()
  if (viewMode.value === 'horizontal') {
    if (normalizedPage.value <= 1) return
    displayPage.value = normalizedPage.value - 2
  } else {
    if (displayPage.value <= 1) return
    displayPage.value--
  }
  preloadImages((normalizedPage.value - 1))
  saveProgress()
  scrollToCurrentPage()
}

const nextPage = () => {
  resetCountdown()
  if (viewMode.value === 'horizontal') {
    if (normalizedPage.value + 1 >= totalPages.value) return
    displayPage.value = normalizedPage.value + 2
  } else {
    if (displayPage.value >= totalPages.value) return
    displayPage.value++
  }
  preloadImages((normalizedPage.value - 1))
  saveProgress()
  scrollToCurrentPage()
}

const goToPage = (page) => {
  resetCountdown()
  if (page < 1 || page > totalPages.value) return
  displayPage.value = page
  showPageSelect.value = false
  preloadImages((normalizedPage.value - 1))
  saveProgress()

  // 竖屏模式：滚动到对应图片
  if (viewMode.value === 'vertical') {
    setTimeout(() => {
      const target = document.querySelector(`.v-page:nth-child(${page})`)
      if (target) {
        const rect = target.getBoundingClientRect()
        const scrollY = window.scrollY + rect.top
        window.scrollTo({ top: scrollY, behavior: 'instant' })
      }
    }, 50)
  }
}

// ========== 横屏点击 ==========
const onImageAreaClick = (e) => {
  if (viewMode.value !== 'horizontal') return
  const rect = e.currentTarget.getBoundingClientRect()
  const x = e.clientX - rect.left
  if (x < rect.width / 2) {
    prevPage()
  } else {
    nextPage()
  }
}

// ========== 竖屏滚动记录 ==========
const onScroll = () => {
  if (viewMode.value !== 'vertical' || !book.value || isDragging.value) return
  const container = document.querySelector('.detail-scroll-container')
  if (!container) return
  const imgs = container.querySelectorAll('.v-page')
  for (let i = 0; i < imgs.length; i++) {
    const rect = imgs[i].getBoundingClientRect()
    if (rect.top >= 0 && rect.top < window.innerHeight / 2) {
      displayPage.value = i + 1
      localStorage.setItem(`${PROGRESS_KEY_PREFIX}${book.value.id}`, String(i + 1))
      break
    }
  }
}

// ========== 切换模式 ==========
const toggleViewMode = () => {
  stopAutoPlay()
  viewMode.value = viewMode.value === 'horizontal' ? 'vertical' : 'horizontal'
  localStorage.setItem(VIEW_MODE_KEY, viewMode.value)
}

// ========== 跳到顶部 ==========
const scrollToTop = () => {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// ========== 键盘控制 ==========
const onKeydown = (e) => {
  if (showPageSelect.value) return
  if (e.key === 'ArrowLeft') prevPage()
  if (e.key === 'ArrowRight') nextPage()
}

// ========== 加载漫画 ==========
const loadBook = async () => {
  stopAutoPlay()
  loading.value = true
  error.value = ''
  images.value = []
  preloadCache.clear()

  try {
    const id = route.params.id
    const result = await BookGet(String(id))
    if (!result) {
      error.value = '未找到该漫画'
      return
    }
    book.value = result

    // 加载章节信息（用于上一章/下一章按钮）
    if (result.parent >= 1) {
      try {
        const list = await GetChapters(result.jmid, result.parent)
        chapters.value = list || []
      } catch (e) {
        console.error('加载章节失败:', e)
      }
    } else {
      chapters.value = []
    }

    const imgs = await BookGetImage(Number(id), 1)
    if (!imgs || imgs.length === 0) {
      error.value = '该漫画没有图片'
      return
    }
    images.value = imgs.filter(Boolean)
    totalPages.value = images.value.length

    // 恢复阅读位置：URL 参数决定起始页，阅读页只记录不自动跳转
    const urlPage = parseInt(route.query.page, 10)
    let startPage = urlPage > 0 ? urlPage : 1
    if (startPage > totalPages.value) startPage = 1
    displayPage.value = startPage

    // 恢复横竖屏模式（全局统一）
    const savedViewMode = localStorage.getItem(VIEW_MODE_KEY)
    if (savedViewMode === 'horizontal' || savedViewMode === 'vertical') {
      viewMode.value = savedViewMode
    }

    preloadImages(normalizedPage.value - 1)

    // 缓存了自动播放状态 → 自动开始播放
    if (localStorage.getItem(AUTOPLAY_KEY) === '1') {
      setTimeout(() => startAutoPlay(), 300)
    }

    // 延迟绑定滚动事件（滚动在 window 上）
    setTimeout(() => {
      window.addEventListener('scroll', onScroll, { passive: true })
    }, 100)
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

// ========== 生命周期 ==========
onMounted(() => {
  loadBook()
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('mousemove', onVerticalMousemove)
  window.addEventListener('mouseup', onVerticalMouseup)
})

onUnmounted(() => {
  stopAutoPlay()
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('mousemove', onVerticalMousemove)
  window.removeEventListener('mouseup', onVerticalMouseup)
  preloadCache.clear()
  // 退出页面时保存阅读进度到数据库
  if (book.value) {
    BookUpdateProgress(Number(book.value.id), displayPage.value).catch(() => { })
  }
})

watch(() => route.params.id, loadBook)
</script>

<template>
  <div class="detail-page">
    <!-- 加载状态 -->
    <div v-if="loading" class="detail-loading">
      <div class="loading-spinner"></div>
      <p>加载中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="detail-error">
      <p>{{ error }}</p>
      <button class="retry-btn" @click="loadBook">重试</button>
    </div>

    <!-- 内容 -->
    <template v-else>
      <!-- 横屏翻页模式 -->
      <div v-if="viewMode === 'horizontal'" class="horizontal-view" @click="onImageAreaClick">
        <div class="h-page h-page-left">
          <img v-if="currentPages[0]" :src="imageUrl(currentPages[0])" alt="" />
          <div v-else class="h-page-empty"></div>
        </div>
        <div class="h-divider"></div>
        <div class="h-page h-page-right">
          <img v-if="currentPages[1]" :src="imageUrl(currentPages[1])" alt="" />
          <div v-else class="h-page-empty"></div>
        </div>
      </div>

      <!-- 竖屏滚动模式 -->
      <div v-else ref="verticalContainer" class="vertical-view detail-scroll-container" tabindex="0"
        @mousedown="onVerticalMousedown">
        <div v-for="(img, idx) in images" :key="idx" class="v-page">
          <img :src="imageUrl(img)" :alt="`第 ${idx + 1} 页`" loading="lazy" />
        </div>
      </div>

      <!-- 右下角浮动按钮组 -->
      <div class="floating-btns">
        <button v-if="viewMode === 'vertical'" class="float-btn" title="跳到顶部" @click="scrollToTop">↑</button>
        <button v-if="viewMode === 'vertical'" class="float-btn" :class="{ 'auto-play-running': autoPlaying, 'auto-play-countdown': countdown !== null }" title="自动播放" @click="toggleAutoPlay">
          {{ countdown !== null ? countdown : (autoPlaying ? '⏸' : '▶') }}
        </button>
        <button v-if="hasPrevChapter" class="float-btn" title="上一章" @click="prevChapter">◀</button>
        <button v-if="hasNextChapter" class="float-btn" title="下一章" @click="nextChapter">▶</button>
        <div class="float-btn-group">
          <button class="float-btn" title="切换页码" @click="showPageSelect = !showPageSelect">
            {{ pageText }}
          </button>
          <div v-if="showPageSelect" class="page-select-dropdown">
            <div class="page-select-list">
              <button v-for="p in pageList" :key="p" class="page-select-item" :class="{ active: p === displayPage }"
                @click="goToPage(p)">
                {{ p }}
              </button>
            </div>
          </div>
        </div>
        <button class="float-btn" title="返回简介" @click="router.push(`/resume/${book.id}`)">
          ←
        </button>
        <button class="float-btn" title="切换观看模式" @click="toggleViewMode">
          {{ viewMode === 'horizontal' ? '竖' : '横' }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.detail-page {
  min-height: 100vh;
  background: var(--app-bg);
  color: var(--app-text);
}

/* ========== 加载 / 错误 ========== */
.detail-loading,
.detail-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  gap: 16px;
}

.detail-error {
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
}

.retry-btn:hover {
  background: #e74c3c;
  color: #fff;
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

/* ========== 横屏翻页模式 ========== */
.horizontal-view {
  display: flex;
  height: 100vh;
  cursor: pointer;
  user-select: none;
}

.h-page {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.h-page img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  display: block;
}

.h-page-empty {
  width: 100%;
  height: 100%;
  background: #1a1a1a;
}

.h-divider {
  width: 2px;
  background: var(--sidebar-border);
  flex-shrink: 0;
}

/* ========== 竖屏滚动模式 ========== */
.vertical-view {
  min-height: 100vh;
  padding: 20px 0;
  outline: none;
}

.v-page {
  display: flex;
  justify-content: center;
  margin-bottom: 8px;
}

.v-page img {
  max-width: 90%;
  display: block;
}

/* ========== 右下角浮动按钮 ========== */
.floating-btns {
  position: fixed;
  bottom: 155px;
  right: 5px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 100;
}

.float-btn-group {
  position: relative;
}

.float-btn {
  width: 44px;
  height: 44px;
  border: 1px solid var(--switch-border);
  border-radius: 8px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  outline: none;
  transition: background-color 0.12s ease, border-color 0.12s ease;
}

.float-btn:hover {
  background: var(--switch-bg);
  border-color: var(--app-text);
}

.float-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.float-btn.auto-play-running {
  background: #27ae60;
  border-color: #27ae60;
  color: #fff;
}

.float-btn.auto-play-countdown {
  background: #e67e22;
  border-color: #e67e22;
  color: #fff;
}

/* 页码选择下拉 */
.page-select-dropdown {
  position: absolute;
  bottom: 0;
  right: 52px;
  background: var(--sidebar-bg);
  border: 1px solid var(--sidebar-border);
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.page-select-list {
  max-height: 300px;
  overflow-y: auto;
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 2px;
  padding: 6px;
}

.page-select-item {
  width: 36px;
  height: 36px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: transparent;
  color: var(--app-text);
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-select-item:hover {
  background: var(--switch-bg);
  border-color: var(--switch-border);
}

.page-select-item.active {
  background: var(--app-text);
  color: var(--app-bg);
  font-weight: 600;
}
</style>
