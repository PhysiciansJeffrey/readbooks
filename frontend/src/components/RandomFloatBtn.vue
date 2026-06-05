<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { BookList } from '../../bindings/ReadBooks/appservice'
import { useRouter } from 'vue-router'

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
    const [books] = await BookList(1, 10000)
    if (!books || books.length === 0) return
    const randomBook = books[Math.floor(Math.random() * books.length)]
    router.push(`/resume/${randomBook.id}`)
  } catch (e) {
    console.error('随机跳转失败:', e)
  }
}
</script>

<template>
  <button v-if="visible" class="random-float-btn" title="随机观看" @click="randomRead">?</button>
</template>

<style scoped>
.random-float-btn {
  position: fixed;
  bottom: 24px;
  right: 24px;
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
  z-index: 100;
  transition: background-color 0.12s ease, border-color 0.12s ease;
}

.random-float-btn:hover {
  background: var(--switch-bg);
  border-color: var(--app-text);
}

.random-float-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--focus-ring);
}
</style>
