<script setup>
import { computed } from 'vue'

const props = defineProps({
  currentPage: { type: Number, required: true },
  totalPages: { type: Number, required: true },
  pageSize: { type: Number, required: true },
  loading: { type: Boolean, default: false },
  minPageSize: { type: Number, default: 10 },
  maxPageSize: { type: Number, default: 100 },
})

const emit = defineEmits(['goto', 'prev', 'next', 'update:pageSize'])

// 生成要显示的页码数组（最多显示5页，当前页居中）
const displayPages = computed(() => {
  const total = props.totalPages
  const cur = props.currentPage
  const pages = []

  let start = Math.max(1, cur - 2)
  let end = Math.min(total, start + 4)

  if (end - start < 4) {
    start = Math.max(1, end - 4)
  }

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }
  return pages
})

const onJumpPage = () => {
  let val = parseInt(jumpPageRef.value, 10)
  if (isNaN(val)) return
  if (val < 1) val = 1
  if (val > props.totalPages) val = props.totalPages
  emit('goto', val)
}

const onPageSizeChange = (delta) => {
  let val = props.pageSize + delta
  if (val < props.minPageSize) val = props.minPageSize
  if (val > props.maxPageSize) val = props.maxPageSize
  if (val === props.pageSize) return
  emit('update:pageSize', val)
}

// 用于 v-model 的页码输入框
import { ref, watch } from 'vue'
const jumpPageRef = ref(props.currentPage)
watch(() => props.currentPage, (v) => { jumpPageRef.value = v })
</script>

<template>
  <nav class="pagination" aria-label="翻页">
    <button class="page-btn page-nav" :disabled="currentPage <= 1 || loading" aria-label="首页"
      @click="emit('goto', 1)">&laquo;</button>

    <button class="page-btn page-nav" :disabled="currentPage <= 1 || loading" aria-label="上一页"
      @click="emit('prev')">&lsaquo;</button>

    <template v-for="p in displayPages" :key="p">
      <input
        v-if="p === currentPage"
        v-model.number="jumpPageRef"
        type="number"
        min="1"
        :max="totalPages"
        class="page-input"
        aria-label="当前页"
        @keyup.enter="onJumpPage"
        @blur="onJumpPage"
      />
      <button v-else class="page-btn" @click="emit('goto', p)">{{ p }}</button>
    </template>

    <button class="page-btn page-nav" :disabled="currentPage >= totalPages || loading" aria-label="下一页"
      @click="emit('next')">&rsaquo;</button>

    <button class="page-btn page-nav" :disabled="currentPage >= totalPages || loading" aria-label="末页"
      @click="emit('goto', totalPages)">&raquo;</button>

    <!-- 每页数量 -->
    <div class="page-size-control">
      <button class="page-btn page-size-btn" :disabled="pageSize <= minPageSize || loading"
        aria-label="减少每页数量" @click="onPageSizeChange(-1)">-</button>
      <input
        :value="pageSize"
        class="page-size-input"
        type="number"
        :min="minPageSize"
        :max="maxPageSize"
        aria-label="每页显示数量"
        @keyup.enter="(e) => emit('update:pageSize', parseInt(e.target.value) || pageSize)"
      />
      <button class="page-btn page-size-btn" :disabled="pageSize >= maxPageSize || loading"
        aria-label="增加每页数量" @click="onPageSizeChange(1)">+</button>
    </div>
  </nav>
</template>

<style scoped>
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

.page-btn {
  display: inline-flex;
  justify-content: center;
  line-height: 35px;
  min-width: 36px;
  height: 36px;
  border: 1px solid gray;
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  cursor: pointer;
  transition: background-color 0.12s ease, border-color 0.12s ease, opacity 0.12s ease;
  outline: none;
}

.page-btn:hover:not(:disabled) {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.page-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.page-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.page-nav {
  padding: 0 10px;
  font-size: 33px;
  line-height: 30px;
}

.page-input {
  width: auto;
  min-width: 36px;
  height: 36px;
  border: 1px solid gray;
  border-radius: 6px;
  background: var(--app-text);
  color: var(--app-bg);
  font-size: 14px;
  font-weight: 600;
  text-align: center;
  outline: none;
  transition: background-color 0.12s ease, border-color 0.12s ease, opacity 0.12s ease;
}

.page-input:focus {
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.page-input::-webkit-outer-spin-button,
.page-input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.page-size-control {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: 8px;
}

.page-size-btn {
  min-width: 30px;
  height: 36px;
  font-size: 18px;
  line-height: 30px;
}

.page-size-input {
  width: 48px;
  height: 36px;
  border: 1px solid gray;
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  text-align: center;
  outline: none;
  transition: background-color 0.12s ease, border-color 0.12s ease;
}

.page-size-input:focus {
  border-color: var(--app-text);
  background: var(--switch-bg);
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.page-size-input::-webkit-outer-spin-button,
.page-size-input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
</style>
