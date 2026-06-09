<script setup>
import { ref, inject, onMounted, computed, watch } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { BookGet, BookGetTags, TagCreate, BookSetTags, BookDelete, BookDeleteWithFiles, GetChapters } from '@/api'

const refreshHome = inject('refreshHome')
const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref('')
const book = ref(null)
const tags = ref([])
const newTag = ref('')
const showTagInput = ref(false)

// 章节目录
const chapters = ref([])
const chaptersLoading = ref(false)

const isMultiChapter = computed(() => {
  return book.value && book.value.parent >= 1
})

const loadBook = async () => {
  loading.value = true
  error.value = ''
  book.value = null
  tags.value = []
  chapters.value = []

  try {
    const id = parseInt(route.params.id, 10)
    if (isNaN(id)) {
      error.value = '无效的书籍 ID'
      return
    }

    const result = await BookGet(String(id))
    if (!result) {
      error.value = '未找到该漫画'
      return
    }
    book.value = {
      ...result,
      coverSrc: result.cover_url ? `/api/image?p=${encodeURIComponent(result.cover_url)}` : '',
    }
    tags.value = result.tags || []

    // 多章节子漫画 → 加载章节目录
    if (result.parent >= 1) {
      loadChapters(result.jmid, result.parent)
    }
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

const loadChapters = async (jmid, parent) => {
  chaptersLoading.value = true
  try {
    const list = await GetChapters(jmid, parent)
    chapters.value = list || []
    setTimeout(() => {
      const currentEl = document.querySelector('.chapter-item.chapter-current')
      if (currentEl) {
        currentEl.scrollIntoView({ block: 'center', behavior: 'instant' })
      }
    }, 50)
  } catch (e) {
    console.error('加载章节目录失败:', e)
  } finally {
    chaptersLoading.value = false
  }
}

const addTag = async () => {
  const tagName = newTag.value.trim()
  if (!tagName) return
  if (!book.value) return

  // 标签已关联到当前书籍 → 直接返回
  if (tags.value.some((t) => t.name === tagName)) {
    newTag.value = ''
    showTagInput.value = false
    return
  }

  try {
    // 创建标签（已存在则返回已有 ID）
    const tagID = await TagCreate(tagName, '')
    // 关联到当前书籍
    const tagIDs = [...tags.value.map((t) => t.id), tagID]
    await BookSetTags(book.value.id, tagIDs)
    // 重新加载标签
    const tagList = await BookGetTags(book.value.id)
    tags.value = (tagList || []).filter(Boolean)
  } catch (e) {
    console.error('添加标签失败:', e)
  }

  newTag.value = ''
  showTagInput.value = false
}

onMounted(loadBook)

// 路由参数变化时重新加载（点击章节目录切换漫画）
watch(() => route.params.id, (newId) => {
  if (newId) loadBook()
})

// --- 删除漫画 ---
const deleteMode = ref(null) // null | 'record' | 'files'

const requestDelete = (mode) => {
  deleteMode.value = mode
}

const cancelDelete = () => {
  deleteMode.value = null
}

const confirmDelete = async () => {
  if (!book.value || !deleteMode.value) return
  try {
    if (deleteMode.value === 'files') {
      await BookDeleteWithFiles(Number(book.value.id))
    } else {
      await BookDelete(Number(book.value.id))
    }
    deleteMode.value = null
    refreshHome?.()
    router.back()
  } catch (e) {
    console.error('删除失败:', e)
    deleteMode.value = null
  }
}
</script>

<template>
  <div class="resume-page">
    <!-- 加载中 -->
    <div v-if="loading" class="resume-loading">
      <div class="loading-spinner"></div>
      <p>加载中...</p>
    </div>

    <!-- 错误 / 未找到 -->
    <div v-else-if="error || !book" class="resume-not-found">
      <p>{{ error || '未找到该漫画' }}</p>
    </div>

    <!-- 内容 -->
    <div v-else class="resume-box">
      <div class="resume-cover">
        <img v-if="book.coverSrc" :src="book.coverSrc" :alt="book.title" />
        <div v-else class="resume-cover-placeholder">暂无封面</div>
      </div>
      <div class="resume-info">
        <h1 class="resume-title">{{ book.title }}</h1>
        <p class="resume-author">{{ book.author }}</p>

        <!-- 阅读进度 -->
        <p class="resume-progress" v-if="book.total_pages > 0">
          阅读进度：{{ book.current_page }} / {{ book.total_pages }} 页
        </p>

        <!-- 状态 -->
        <p class="resume-status" v-if="book.status">
          状态：{{ book.status }}
        </p>

        <!-- 标签 -->
        <div class="resume-tags">
          <button v-for="tag in tags" :key="tag.id" class="resume-tag"
            @click="$router.push(`/search/${encodeURIComponent(tag.name)}`)">
            {{ tag.name }}
          </button>
          <input v-if="showTagInput" v-model="newTag" class="resume-tag-input" type="text" placeholder="新标签"
            @keyup.enter="addTag" @blur="addTag" @keyup.esc="showTagInput = false; newTag = ''" />
          <button v-else class="resume-tag resume-tag-add" @click="showTagInput = true">+</button>
        </div>

        <!-- 操作按钮 -->
        <div class="resume-actions">
          <button v-if="book.current_page > 0" class="resume-btn resume-btn-primary"
            @click="$router.push(`/detail/${book.id}?page=${book.current_page}`)">
            继续阅读
          </button>
          <button class="resume-btn resume-btn-primary"
            @click="$router.push(`/detail/${book.id}?page=1`)">
            开始阅读
          </button>
          <button class="resume-btn">收藏</button>
          <button class="resume-btn" @click="requestDelete('record')">仅删记录</button>
          <button class="resume-btn resume-btn-danger" @click="requestDelete('files')">删除记录+源文件</button>
        </div>

        <!-- 章节目录（多章节子漫画） -->
        <div v-if="isMultiChapter" class="resume-chapters">
          <h3 class="resume-chapters-title">章节目录</h3>
          <div v-if="chaptersLoading" class="chapters-loading">加载中...</div>
          <div v-else-if="chapters.length === 0" class="chapters-empty">暂无章节</div>
          <div v-else class="chapters-list">
            <div
              v-for="ch in chapters"
              :key="ch.id"
              class="chapter-item"
              :class="{ 'chapter-current': ch.id === book.id }"
              @click="router.push(`/resume/${ch.id}`)"
            >
              <span class="chapter-title">{{ ch.title }}</span>
              <span class="chapter-progress">{{ ch.current_page }}/{{ ch.total_pages }}页</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 删除确认弹窗 -->
    <div v-if="deleteMode" class="dialog-overlay" @click.self="cancelDelete">
      <div class="dialog-box">
        <p class="dialog-text" v-if="deleteMode === 'files'">确认删除《{{ book.title }}》？此操作将同时删除电脑上的源文件夹，不可恢复。</p>
        <p class="dialog-text" v-else>确认仅删除《{{ book.title }}》的记录？源文件夹将保留在电脑上。</p>
        <div class="dialog-btns">
          <button class="dialog-btn dialog-btn-cancel" @click="cancelDelete">取消</button>
          <button class="dialog-btn dialog-btn-confirm" @click="confirmDelete">确认删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

.resume-page {
  display: flex;
  gap: 32px;
  padding: 32px;
  margin: 0 auto;
}
.resume-box{
  width: 100%;
  display: flex;
  gap:30px;
}
@media (max-width: 740px) {
  .resume-box {
    display: ruby;
    /* gap: 16px; */
  }

  .resume-cover {
    width: 100%;
    min-width: 0;
    max-height: 300px;
  }

  .resume-cover img {
    height: auto;
    max-height: 300px;
  }

  .resume-info {
    width: 100%;
  }
  .resume-info div{
    justify-content: center;
  }
}


.resume-cover {
  flex-shrink: 0;
  width: 240px;
  min-width: 40%;
  border-radius: 8px;
  overflow: hidden;
  background: var(--sidebar-border);
}

.resume-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.resume-cover-placeholder {
  width: 100%;
  aspect-ratio: 3 / 4;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--muted-text);
  font-size: 14px;
}

.resume-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.resume-title {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
  color: var(--app-text);
  line-height: 1.3;
}

.resume-author {
  margin: 0;
  font-size: 16px;
  color: var(--muted-text);
}

.resume-progress,
.resume-status {
  margin: 0;
  font-size: 14px;
  color: var(--muted-text);
}

.resume-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.resume-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 999px;
  background: var(--switch-border);
  color: var(--app-text);
  font-size: 12px;
  line-height: 1.6;
  border: none;
  cursor: pointer;
  font-family: inherit;
}

.resume-tag-add {
  background: transparent;
  border: 1px dashed var(--switch-border);
  cursor: pointer;
  font-size: 14px;
  line-height: 1.4;
  padding: 1px 10px;
  font-weight: 600;
}

.resume-tag-add:hover {
  border-color: var(--app-text);
  background: var(--switch-border);
}

.resume-actions {
  display: flex;
  gap: 10px;
  margin-top: 20px;
}

.resume-btn {
  padding: 8px 20px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  font-size: 14px;
  cursor: pointer;
  outline: none;
}

.resume-btn:hover {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.resume-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.resume-btn-primary {
  background-color: #fa9292;
  border-width: 2px;
  border-color: #ff0000;
}

.resume-btn-danger {
  background: transparent;
  border-color: #e74c3c;
  color: #e74c3c;
}

.resume-btn-danger:hover {
  background: #e74c3c;
  color: #fff;
}

/* 弹窗 */
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.dialog-box {
  background: var(--sidebar-bg);
  border: 1px solid var(--sidebar-border);
  border-radius: 8px;
  padding: 24px;
  min-width: 300px;
  max-width: 90vw;
}

.dialog-text {
  margin: 0 0 20px;
  font-size: 14px;
  color: var(--app-text);
  text-align: center;
}

.dialog-btns {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.dialog-btn {
  padding: 8px 20px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  cursor: pointer;
  outline: none;
}

.dialog-btn:hover {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.dialog-btn-cancel {
  border-color: var(--switch-border);
}

.dialog-btn-confirm {
  background: #e74c3c;
  border-color: #e74c3c;
  color: #fff;
}

.dialog-btn-confirm:hover {
  background: #c0392b;
  border-color: #c0392b;
}

.resume-tag-input {
  display: inline-block;
  width: 80px;
  padding: 2px 8px;
  border: 1px solid var(--switch-border);
  border-radius: 999px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 12px;
  line-height: 1.6;
  outline: none;
  text-align: center;
}

.resume-tag-input:focus {
  border-color: var(--app-text);
  box-shadow: 0 0 0 2px var(--focus-ring);
}

/* 加载状态 */
.resume-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  gap: 16px;
  color: var(--muted-text);
  width: 100%;
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

/* 未找到 */
.resume-not-found {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  color: var(--muted-text);
  font-size: 16px;
  width: 100%;
}

/* 章节目录 */
.resume-chapters {
  margin-top: 24px;
  border-top: 1px solid var(--sidebar-border);
  padding-top: 16px;
}

.resume-chapters-title {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 600;
  color: var(--app-text);
}

.chapters-loading,
.chapters-empty {
  font-size: 14px;
  color: var(--muted-text);
  padding: 8px 0;
}

.chapters-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 360px;
  overflow-y: auto;
}

.chapter-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
  background: transparent;
}

.chapter-item:hover {
  background: var(--sidebar-border);
}

.chapter-item.chapter-current {
  background: var(--switch-bg);
  border: 1px solid var(--switch-border);
  font-weight: 600;
}

.chapter-title {
  font-size: 14px;
  color: var(--app-text);
}

.chapter-progress {
  font-size: 12px;
  color: var(--muted-text);
  white-space: nowrap;
}
</style>
