<script setup>
import { useRouter } from 'vue-router'

defineProps({
  comics: { type: Array, default: () => [] },
})

const router = useRouter()

// --- 封面 URL 缓存 ---
const coverCache = new Map()
const loadCover = (path) => {
  if (!path) return ''
  if (coverCache.has(path)) return coverCache.get(path)
  const url = `/api/image?p=${encodeURIComponent(path)}`
  coverCache.set(path, url)
  return url
}

const goToSearch = (tagName) => {
  router.push(`/search/${encodeURIComponent(tagName)}`)
}
</script>

<template>
  <div class="comic-grid">
    <div
      v-for="comic in comics"
      :key="comic.id"
      class="comic-card"
    >
      <router-link
        class="comic-card-main"
        :to="`/resume/${comic.id}`"
        :aria-label="comic.title"
      >
        <div class="comic-cover">
          <img :src="loadCover(comic.cover_url)" :alt="comic.title" loading="lazy" />
          <div class="comic-cover-overlay"></div>
        </div>
        <div class="comic-info">
          <h3 class="comic-title">{{ comic.title }}</h3>
          <p class="comic-author">{{ comic.author }}</p>
        </div>
      </router-link>
      <div class="comic-tags" v-if="comic.tags && comic.tags.length">
        <button v-for="tag in comic.tags" :key="tag.id" class="comic-tag"
          @click="goToSearch(tag.name)">{{ tag.name }}</button>
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

.comic-tag:hover {
  background: var(--app-text);
  color: var(--app-bg);
}
</style>
