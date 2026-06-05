<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { BookDeleteWithFiles } from '../../bindings/ReadBooks/appservice'

const emit = defineEmits(['refresh'])

defineProps({
  comics: { type: Array, default: () => [] },
})

const router = useRouter()

// --- 封面 URL 缓存 ---
const coverCache = new Map()
const loadCover = (path) => {
  if (!path) return ''
  if (coverCache.has(path)) return coverCache.get(path)
  const url = `/api/image?cover=1&p=${encodeURIComponent(path)}`
  coverCache.set(path, url)
  return url
}

const goToSearch = (tagName) => {
  router.push(`/search/${encodeURIComponent(tagName)}`)
}

// --- 批量删除 ---
const deleteMode = ref(false)
const selectedIds = ref(new Set())
const deleting = ref(false)

const toggleDeleteMode = () => {
  deleteMode.value = !deleteMode.value
  selectedIds.value.clear()
}

const toggleSelect = (id) => {
  if (selectedIds.value.has(id)) {
    selectedIds.value.delete(id)
  } else {
    selectedIds.value.add(id)
  }
}

const confirmDelete = async () => {
  if (selectedIds.value.size === 0 || deleting.value) return
  deleting.value = true
  try {
    for (const id of selectedIds.value) {
      await BookDeleteWithFiles(id)
    }
    selectedIds.value.clear()
    deleteMode.value = false
    emit('refresh')
  } catch (e) {
    console.error('删除失败:', e)
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <!-- 删除操作栏 -->
  <div v-if="deleteMode" class="delete-bar">
    <span class="delete-bar-info">已选 {{ selectedIds.size }} 项</span>
    <div class="delete-bar-btns">
      <button class="delete-btn delete-btn-confirm" :disabled="selectedIds.size === 0 || deleting" @click="confirmDelete">
        {{ deleting ? '删除中...' : '确认删除' }}
      </button>
      <button class="delete-btn delete-btn-cancel" @click="toggleDeleteMode">取消</button>
    </div>
  </div>

  <button class="delete-mode-btn" @click="toggleDeleteMode">
    {{ deleteMode ? '退出删除' : '删除' }}
  </button>

  <div class="comic-grid">
    <div
      v-for="comic in comics"
      :key="comic.id"
      class="comic-card"
      :class="{ 'comic-card-selected': selectedIds.has(comic.id) }"
      @click="deleteMode && toggleSelect(comic.id)"
    >
      <div v-if="deleteMode" class="comic-select">
        <input type="checkbox" :checked="selectedIds.has(comic.id)" @click.stop="toggleSelect(comic.id)" />
      </div>
      <div v-if="deleteMode" class="comic-card-main" @click.stop>
        <div class="comic-cover">
          <img :src="loadCover(comic.cover_url)" :alt="comic.title" loading="lazy" />
          <div class="comic-cover-overlay"></div>
        </div>
        <div class="comic-info">
          <h3 class="comic-title">{{ comic.title }}</h3>
          <p class="comic-author">{{ comic.author }}</p>
          <span class="comic-pages">{{ comic.total_pages }} 页</span>
        </div>
      </div>
      <router-link
        v-else
        class="comic-card-main"
        :to="`/resume/${comic.id}`"
        :aria-label="comic.title"
        @click.stop
      >
        <div class="comic-cover">
          <img :src="loadCover(comic.cover_url)" :alt="comic.title" loading="lazy" />
          <div class="comic-cover-overlay"></div>
        </div>
        <div class="comic-info">
          <h3 class="comic-title">{{ comic.title }}</h3>
          <p class="comic-author">{{ comic.author }}</p>
          <span class="comic-pages">{{ comic.total_pages }} 页</span>
        </div>
      </router-link>
      <div class="comic-tags" v-if="!deleteMode && comic.tags && comic.tags.length">
        <button v-for="tag in comic.tags" :key="tag.id" class="comic-tag"
          @click.stop="goToSearch(tag.name)">{{ tag.name }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.comic-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 25px;
}

.comic-card {
  display: flex;
  flex-direction: column;
  border-radius: 8px;
  overflow: hidden;
  background: var(--sidebar-bg);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  outline: none;
  text-decoration: none;
  color: inherit;
}

.comic-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.12);
}

.comic-card:focus-visible {
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.comic-card-main {
  display: flex;
  flex-direction: column;
  text-decoration: none;
  color: inherit;
  cursor: pointer;
}

.comic-cover {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  overflow: hidden;
  background: var(--sidebar-border);
}

.comic-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.comic-cover-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to bottom, transparent 60%, rgba(0, 0, 0, 0.12));
  pointer-events: none;
}

.comic-info {
  padding: 8px 10px 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.comic-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
  line-height: 1.3;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.comic-author {
  margin: 0;
  font-size: 12px;
  color: var(--muted-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.comic-tags {
  margin-bottom: 10px;
  margin-left: 5px;
  display: flex;
  flex-wrap: nowrap;
  gap: 4px;
  margin-top: 4px;
  overflow-x: auto;
  scrollbar-width: thin;
  scrollbar-color: var(--switch-border) transparent;
}

.comic-tags::-webkit-scrollbar {
  height: 5px;
}

.comic-tags::-webkit-scrollbar-track {
  background: transparent;
}

.comic-tags::-webkit-scrollbar-thumb {
  background: var(--switch-border);
  border-radius: 3px;
}



.comic-tag {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 999px;
  border: none;
  background: var(--switch-border);
  color: var(--muted-text);
  font-size: 10px;
  line-height: 1.5;
  white-space: nowrap;
  cursor: pointer;
  font-family: inherit;
}

/* 删除模式 */
.delete-mode-btn {
  margin-bottom: 16px;
  padding: 6px 16px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 13px;
  cursor: pointer;
  transition: border-color 0.12s ease, background-color 0.12s ease;
  outline: none;
}

.delete-mode-btn:hover {
  border-color: #e74c3c;
  color: #e74c3c;
}

.delete-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  padding: 10px 14px;
  border: 1px solid #e74c3c;
  border-radius: 6px;
  background: rgba(231, 76, 60, 0.08);
}

.delete-bar-info {
  font-size: 13px;
  color: var(--app-text);
}

.delete-bar-btns {
  display: flex;
  gap: 8px;
}

.delete-btn {
  padding: 4px 14px;
  border: 1px solid var(--switch-border);
  border-radius: 4px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  transition: border-color 0.12s ease, background-color 0.12s ease;
  outline: none;
}

.delete-btn:hover:not(:disabled) {
  border-color: var(--app-text);
}

.delete-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.delete-btn-confirm {
  border-color: #e74c3c;
  color: #e74c3c;
}

.delete-btn-confirm:hover:not(:disabled) {
  background: #e74c3c;
  color: #fff;
}

/* 卡片选中状态 */
.comic-card-selected {
  box-shadow: 0 0 0 2px #e74c3c;
}

.comic-card-selected .comic-cover-overlay {
  background: rgba(231, 76, 60, 0.25);
}

.comic-select {
  position: absolute;
  top: 6px;
  left: 6px;
  z-index: 2;
}

.comic-select input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
  accent-color: #e74c3c;
}

.comic-pages {
  font-size: 11px;
  color: var(--muted-text);
  opacity: 0.7;
}

.comic-tag:hover {
  background: var(--app-text);
  color: var(--app-bg);
}
</style>
