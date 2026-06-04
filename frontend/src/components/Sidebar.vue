<script setup>
import { computed, inject, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Window } from '@wailsio/runtime'

const router = useRouter()
const refreshHome = inject('refreshHome')

const THEME_STORAGE_KEY = 'readbooks-theme'
const SIDE_STORAGE_KEY = 'readbooks-sidebar-side'
const SIZE_STORAGE_KEY = 'readbooks-window-size'

const isDark = ref(false)
const isOpen = ref(false)
const isRightSide = ref(false)
const isBorderlessFullscreen = ref(false)

const themeLabel = computed(() => (isDark.value ? '黑色背景' : '白色背景'))
const sideLabel = computed(() => (isRightSide.value ? '右侧' : '左侧'))
const fullscreenLabel = computed(() => (
  isBorderlessFullscreen.value ? '退出无边框全屏' : '无边框全屏'
))

const openSidebar = () => {
  isOpen.value = true
}

const closeSidebar = () => {
  isOpen.value = false
}

const goHome = () => {
  closeSidebar()
  refreshHome?.()
  router.push('/')
}

const refreshFullscreenState = async () => {
  try {
    isBorderlessFullscreen.value = await Window.IsFullscreen()
  } catch {
    isBorderlessFullscreen.value = false
  }
}

const exitBorderlessFullscreen = async () => {
  try {
    const isFullscreen = await Window.IsFullscreen()

    if (!isFullscreen) {
      isBorderlessFullscreen.value = false
      return false
    }

    await Window.UnFullscreen()
    await Window.SetFrameless(false)

    // 退出全屏后恢复之前的窗口尺寸
    const saved = localStorage.getItem(SIZE_STORAGE_KEY)
    if (saved) {
      try {
        const size = JSON.parse(saved)
        if (size && size.width > 0 && size.height > 0) {
          await Window.SetSize(size.width, size.height)
        }
      } catch {}
    }

    isBorderlessFullscreen.value = false
    return true
  } catch (error) {
    console.error('退出无边框全屏失败', error)
    return false
  }
}

const toggleBorderlessFullscreen = async () => {
  try {
    const isFullscreen = await Window.IsFullscreen()

    if (isFullscreen) {
      await exitBorderlessFullscreen()
      return
    }

    await Window.SetFrameless(true)
    await Window.Fullscreen()
    isBorderlessFullscreen.value = true
  } catch (error) {
    console.error('切换无边框全屏失败', error)
  }
}

const handleKeydown = async (event) => {
  if (event.key !== 'Escape') {
    return
  }

  event.preventDefault()

  const didExitFullscreen = await exitBorderlessFullscreen()

  if (!didExitFullscreen) {
    closeSidebar()
  }
}

// --- 自动保存/恢复窗口尺寸 ---
let resizeTimer = null

const saveWindowSize = async () => {
  try {
    const isFullscreen = await Window.IsFullscreen()
    if (isFullscreen) return

    const isMaximised = await Window.IsMaximised()
    if (isMaximised) return

    const size = await Window.Size()
    if (size && size.width > 0 && size.height > 0) {
      localStorage.setItem(SIZE_STORAGE_KEY, JSON.stringify(size))
    }
  } catch (error) {
    console.error('保存窗口尺寸失败', error)
  }
}

const handleResize = () => {
  clearTimeout(resizeTimer)
  resizeTimer = setTimeout(saveWindowSize, 300)
}

const restoreWindowSize = async () => {
  try {
    const saved = localStorage.getItem(SIZE_STORAGE_KEY)
    if (!saved) return

    const size = JSON.parse(saved)
    if (size && size.width > 0 && size.height > 0) {
      await Window.SetSize(size.width, size.height)
    }
  } catch (error) {
    console.error('恢复窗口尺寸失败', error)
  }
}

const applyTheme = () => {
  const theme = isDark.value ? 'dark' : 'light'

  document.documentElement.dataset.theme = theme
  localStorage.setItem(THEME_STORAGE_KEY, theme)
}

watch(isRightSide, (val) => {
  localStorage.setItem(SIDE_STORAGE_KEY, val ? 'right' : 'left')
})

onMounted(() => {
  const savedTheme = localStorage.getItem(THEME_STORAGE_KEY)

  isDark.value = savedTheme === 'dark'
  applyTheme()
  refreshFullscreenState()

  const savedSide = localStorage.getItem(SIDE_STORAGE_KEY)
  isRightSide.value = savedSide === 'right'

  window.addEventListener('keydown', handleKeydown)

  // 恢复上次的窗口尺寸
  restoreWindowSize()
  // 监听窗口尺寸变化自动保存
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', handleResize)
  clearTimeout(resizeTimer)
})

watch(isDark, applyTheme)
</script>

<template>
  <div
    v-show="!isOpen"
    class="sidebar-toggle-zone"
    :class="{ 'is-fullscreen': isBorderlessFullscreen, 'is-right': isRightSide }"
  >
    <button
      class="sidebar-toggle"
      type="button"
      aria-label="展开侧边栏"
      :aria-expanded="isOpen"
      @click="openSidebar"
    >
      <span></span>
      <span></span>
      <span></span>
    </button>
  </div>

  <Transition :name="isRightSide ? 'sidebar-fade-right' : 'sidebar-fade-left'">
    <div v-if="isOpen" class="sidebar-layer" @click.self="closeSidebar">
      <aside class="sidebar" :class="{ 'sidebar-right': isRightSide }" aria-label="侧边栏设置" @click.stop>
        <button class="sidebar-home-btn" type="button" @click="goHome">返回首页</button>
        <section class="settings-list">
          <div class="setting-panel">
            <div>
              <p class="setting-title">背景设置</p>
              <p class="setting-value">{{ themeLabel }}</p>
            </div>

            <label class="theme-switch">
              <input v-model="isDark" type="checkbox" aria-label="切换黑白背景" />
              <span class="switch-track">
                <span class="switch-thumb"></span>
              </span>
            </label>
          </div>

          <div class="setting-panel">
            <div>
              <p class="setting-title">侧边栏位置</p>
              <p class="setting-value">{{ sideLabel }}</p>
            </div>

            <button
              class="side-toggle"
              type="button"
              :aria-pressed="isRightSide"
              aria-label="切换侧边栏左右位置"
              @click="isRightSide = !isRightSide"
            >
              <span class="side-icon" :class="{ 'is-right': isRightSide }"></span>
            </button>
          </div>

          <div class="setting-panel">
            <div>
              <p class="setting-title">窗口设置</p>
              <p class="setting-value">{{ fullscreenLabel }}</p>
            </div>

            <button
              class="window-action"
              type="button"
              :aria-pressed="isBorderlessFullscreen"
              @click="toggleBorderlessFullscreen"
            >
              <span class="fullscreen-icon"></span>
            </button>
          </div>
        </section>
        <router-link to="/setting" class="sidebar-setting-btn" @click="closeSidebar">设置</router-link>
      </aside>
    </div>
  </Transition>
</template>

<style scoped>
.sidebar-toggle-zone {
  position: fixed;
  top: 24px;
  left: 0;
  z-index: 20;
  display: flex;
  width: 44px;
  height: 44px;
  align-items: flex-start;
  justify-content: flex-start;
}

.sidebar-toggle-zone.is-right {
  left: auto;
  right: 0;
  justify-content: flex-end;
}

.sidebar-toggle-zone.is-fullscreen {
  top: 0;
  bottom: 0;
  width: auto;
  height: auto;
  padding-top: 24px;
  flex-direction: column;
}

.sidebar-toggle-zone.is-fullscreen.is-right {
  justify-content: flex-start;
}

.sidebar-toggle {
  display: inline-flex;
  width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 5px;
  margin: 0;
  padding: 0;
  border: 1px solid var(--sidebar-border);
  border-left: 0;
  border-radius: 0 8px 8px 0;
  background: var(--sidebar-bg);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
  cursor: pointer;
  transition: opacity 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
}

.sidebar-toggle-zone.is-right .sidebar-toggle {
  border-left: 1px solid var(--sidebar-border);
  border-right: 0;
  border-radius: 8px 0 0 8px;
}

.sidebar-toggle-zone.is-fullscreen .sidebar-toggle {
  opacity: 0;
  transform: translateX(-34px);
  pointer-events: none;
}

.sidebar-toggle-zone.is-fullscreen:hover .sidebar-toggle,
.sidebar-toggle-zone.is-fullscreen:focus-within .sidebar-toggle {
  opacity: 1;
  transform: translateX(0);
  pointer-events: auto;
}

.sidebar-toggle-zone.is-fullscreen.is-right .sidebar-toggle {
  transform: translateX(34px);
}

.sidebar-toggle-zone.is-fullscreen.is-right:hover .sidebar-toggle,
.sidebar-toggle-zone.is-fullscreen.is-right:focus-within .sidebar-toggle {
  transform: translateX(0);
}

.sidebar-toggle span {
  width: 18px;
  height: 2px;
  border-radius: 999px;
  background: var(--app-text);
}

.sidebar-toggle:focus-visible {
  outline: 2px solid var(--focus-ring);
  outline-offset: 3px;
}

.sidebar-layer {
  position: fixed;
  inset: 0;
  z-index: 30;
  background: rgba(0, 0, 0, 0.42);
}

.sidebar {
  position: absolute;
  top: 0;
  left: 0;
  width: 240px;
  height: 100vh;
  padding: 24px 18px;
  box-sizing: border-box;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--sidebar-border);
  box-shadow: 16px 0 40px rgba(0, 0, 0, 0.24);
  color: var(--app-text);
  text-align: left;
  display: flex;
  flex-direction: column;
}

.sidebar-right {
  left: auto;
  right: 0;
  border-right: 0;
  border-left: 1px solid var(--sidebar-border);
  box-shadow: -16px 0 40px rgba(0, 0, 0, 0.24);
}

/* 左侧滑入动画 */
.sidebar-fade-left-enter-active,
.sidebar-fade-left-leave-active {
  transition: opacity 0.18s ease;
}

.sidebar-fade-left-enter-active .sidebar,
.sidebar-fade-left-leave-active .sidebar {
  transition: transform 0.18s ease;
}

.sidebar-fade-left-enter-from,
.sidebar-fade-left-leave-to {
  opacity: 0;
}

.sidebar-fade-left-enter-from .sidebar,
.sidebar-fade-left-leave-to .sidebar {
  transform: translateX(-100%);
}

/* 右侧滑入动画 */
.sidebar-fade-right-enter-active,
.sidebar-fade-right-leave-active {
  transition: opacity 0.18s ease;
}

.sidebar-fade-right-enter-active .sidebar,
.sidebar-fade-right-leave-active .sidebar {
  transition: transform 0.18s ease;
}

.sidebar-fade-right-enter-from,
.sidebar-fade-right-leave-to {
  opacity: 0;
}

.sidebar-fade-right-enter-from .sidebar,
.sidebar-fade-right-leave-to .sidebar {
  transform: translateX(100%);
}

.settings-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin-top: 12px;
}

.sidebar-home-btn {
  display: block;
  width: 100%;
  padding: 8px 16px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  outline: none;
  text-align: center;
}

.sidebar-home-btn:hover {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.sidebar-home-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.setting-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.setting-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  line-height: 22px;
}

.setting-value {
  margin: 2px 0 0;
  color: var(--muted-text);
  font-size: 12px;
  line-height: 18px;
}

/* 主题开关 */
.theme-switch {
  display: inline-flex;
  cursor: pointer;
}

.theme-switch input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}

.switch-track {
  position: relative;
  display: inline-flex;
  width: 48px;
  height: 28px;
  flex: 0 0 auto;
  border: 1px solid var(--switch-border);
  border-radius: 999px;
  background: var(--switch-bg);
  transition: background-color 0.18s ease, border-color 0.18s ease;
}

.switch-thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--switch-thumb);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.22);
  transition: transform 0.18s ease, background-color 0.18s ease;
}

.theme-switch input:checked + .switch-track .switch-thumb {
  transform: translateX(20px);
}

.theme-switch input:focus-visible + .switch-track {
  outline: 2px solid var(--focus-ring);
  outline-offset: 3px;
}

/* 左右切换按钮 */
.side-toggle {
  position: relative;
  display: inline-flex;
  width: 44px;
  height: 32px;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  margin: 0;
  padding: 0;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--switch-bg);
  color: var(--app-text);
  cursor: pointer;
  transition: border-color 0.18s ease;
}

.side-toggle:hover {
  border-color: var(--app-text);
}

.side-toggle:focus-visible {
  outline: 2px solid var(--focus-ring);
  outline-offset: 3px;
}

.side-toggle[aria-pressed='true'] {
  background: var(--app-text);
  color: var(--app-bg);
}

.side-icon {
  position: relative;
  display: inline-block;
  width: 20px;
  height: 14px;
  border: 2px solid currentColor;
  border-radius: 2px;
  box-sizing: border-box;
  transition: transform 0.18s ease;
}

.side-icon::before {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.6;
}

.side-icon::after {
  content: '';
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.35;
}

.side-icon.is-right {
  transform: scaleX(-1);
}

/* 窗口全屏按钮 */
.window-action {
  position: relative;
  display: inline-flex;
  width: 44px;
  height: 32px;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  margin: 0;
  padding: 0;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--switch-bg);
  color: var(--app-text);
  cursor: pointer;
}

.window-action:hover {
  border-color: var(--app-text);
}

.window-action:focus-visible {
  outline: 2px solid var(--focus-ring);
  outline-offset: 3px;
}

.window-action[aria-pressed='true'] {
  background: var(--app-text);
  color: var(--app-bg);
}

.fullscreen-icon {
  position: relative;
  width: 18px;
  height: 14px;
  border: 2px solid currentColor;
  border-radius: 2px;
  box-sizing: border-box;
}

.fullscreen-icon::before,
.fullscreen-icon::after {
  content: '';
  position: absolute;
  background: currentColor;
}

.fullscreen-icon::before {
  top: 4px;
  left: -5px;
  width: 5px;
  height: 2px;
}

.fullscreen-icon::after {
  right: -5px;
  bottom: 4px;
  width: 5px;
  height: 2px;
}

.sidebar-setting-btn {
  display: block;
  margin-top: auto;
  margin-bottom: 8px;
  padding: 8px 16px;
  text-align: center;
  font-size: 14px;
  color: var(--app-text);
  text-decoration: none;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  transition: border-color 0.12s ease, background-color 0.12s ease;
  cursor: pointer;
}

.sidebar-setting-btn:hover {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

:global(body) {
  background: var(--app-bg);
  color: var(--app-text);
  transition: background-color 0.18s ease, color 0.18s ease;
}
</style>