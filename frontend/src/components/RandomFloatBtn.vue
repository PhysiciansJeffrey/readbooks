<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getUnreadRandomBook, clearRandomCache } from '@/utils/randomReader'

const props = defineProps({
  pages: {
    type: Array,
    default: () => [],
  },
})

const route = useRoute()
const router = useRouter()

const visible = computed(() => {
  if (!props.pages || props.pages.length === 0) return true
  return props.pages.some(p => {
    if (p.endsWith('*')) {
      return route.name?.toString().startsWith(p.slice(0, -1))
    }
    return route.name === p
  })
})

const randomRead = async () => {
  try {
    const book = await getUnreadRandomBook()
    if (!book) return
    const target = route.name === 'Detail' ? `/detail/${book.id}` : `/resume/${book.id}`
    router.push(target)
  } catch (e) {
    console.error('随机跳转失败:', e)
  }
}
</script>

<template>
  <div v-if="visible" class="float-btn-group">
    <button class="random-float-btn" title="随机观看" @click="randomRead">?</button>
    <button class="clear-cache-btn" title="重置随机记录" @click="clearRandomCache">↺</button>
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
