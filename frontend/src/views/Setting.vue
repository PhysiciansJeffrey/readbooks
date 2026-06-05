<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { SelectFolder, AddComic, AddsComic, TagListWithCount, TagUpdate, TagDelete, TagCreate, GetDefaultComicDir, SetDefaultComicDir } from '../../bindings/ReadBooks/appservice'

const router = useRouter()

const paths = ref([])
const loading = ref(false)
const selectOnly = ref(true)

// --- 标签管理 ---
const tags = ref([])
const tagsLoading = ref(false)
const editMode = ref(false)
const editingTagId = ref(null)
const editingName = ref('')
const confirmDeleteId = ref(null)
const newTag = ref('')
const showTagInput = ref(false)

const addTag = async () => {
  const tagName = newTag.value.trim()
  if (!tagName) {
    showTagInput.value = false
    return
  }
  // 已存在同名标签则跳过
  if (tags.value.some((t) => t.name === tagName)) {
    newTag.value = ''
    showTagInput.value = false
    return
  }
  try {
    await TagCreate(tagName, '')
    newTag.value = ''
    showTagInput.value = false
    await fetchTags()
  } catch (e) {
    console.error('添加标签失败:', e)
  }
}

const fetchTags = async () => {
  tagsLoading.value = true
  try {
    tags.value = await TagListWithCount()
  } catch (e) {
    console.error('获取标签失败:', e)
  } finally {
    tagsLoading.value = false
  }
}

const startEditName = (tag) => {
  editingTagId.value = tag.id
  editingName.value = tag.name
  nextTick(() => {
    const input = document.querySelector('.tag-name-input')
    if (input) {
      input.focus()
      input.select()
    }
  })
}

const confirmEditName = async (tag) => {
  const name = editingName.value.trim()
  if (name && name !== tag.name) {
    try {
      await TagUpdate(tag.id, name, tag.color || '')
      tag.name = name
    } catch (e) {
      console.error('修改标签失败:', e)
    }
  }
  editingTagId.value = null
}

const cancelEditName = () => {
  editingTagId.value = null
}

const requestDelete = (tagId) => {
  confirmDeleteId.value = tagId
}

const confirmDelete = async () => {
  const id = confirmDeleteId.value
  if (!id) return
  try {
    await TagDelete(id)
    tags.value = tags.value.filter(t => t.id !== id)
  } catch (e) {
    console.error('删除标签失败:', e)
  }
  confirmDeleteId.value = null
}

const cancelDelete = () => {
  confirmDeleteId.value = null
}

const toggleEditMode = () => {
  editMode.value = !editMode.value
  editingTagId.value = null
}

const pickFolder = async () => {
  try {
    let path = await SelectFolder()
    if (path && !paths.value.find((item) => item.path === path)) {
      paths.value.push({ path, status: 'pending' })
    }
  } catch (e) {
    // 用户取消
  }
}

const pickMultiple = async () => {
  try {
    // if (selectOnly.value) {
    //   selectOnly.value = !selectOnly.value
    //   paths.value = []
    // }
    let path = await SelectFolder()
    if (path && !paths.value.find((item) => item.path === path)) {
      paths.value.push({ path, status: 'pending' })
    }
  } catch (e) {
    console.log(e)
  }
}

const confirmOne = async (index) => {
  let item = paths.value[index]
  item.status = 'loading'
  try {
    // if (selectOnly.value) {
    //   res = await AddComic(item.path)
    // } else {
    // }
    let res = await AddsComic(item.path)
    if (res && res.success) {
      item.status = 'done'
      item.result = res
    } else {
      item.status = 'error'
      item.error = res?.error || '导入失败'
    }
  } catch (e) {
    item.status = 'error'
    item.error = e.message
  }
}

const cancelOne = (index) => {
  paths.value.splice(index, 1)
}

const confirmAll = async () => {
  loading.value = true
  for (let i = 0; i < paths.value.length; i++) {
    if (paths.value[i].status === 'pending') {
      await confirmOne(i)
    }
  }
  loading.value = false
}

const cancelAll = () => {
  paths.value = []
}

const goBack = () => {
  router.back()
}

// --- 下载目录 ---
const downloadDir = ref('')
const savedDir = ref('')
const downloadDirChanged = ref(false)

const loadDownloadDir = async () => {
  try {
    const dir = await GetDownloadDir()
    downloadDir.value = dir
    savedDir.value = dir
  } catch (e) {
    console.error('获取下载目录失败:', e)
  }
}

const changeDownloadDir = async () => {
  try {
    const dir = await SelectFolder()
    if (dir) {
      downloadDir.value = dir
      downloadDirChanged.value = dir !== savedDir.value
    }
  } catch (e) {
    // 用户取消
  }
}

const saveDownloadDir = async () => {
  try {
    await SetDownloadDir(downloadDir.value)
    savedDir.value = downloadDir.value
    downloadDirChanged.value = false
  } catch (e) {
    console.error('保存下载目录失败:', e)
  }
}

const resetDownloadDir = async () => {
  try {
    const dir = await GetDefaultDownloadDir()
    downloadDir.value = dir
    downloadDirChanged.value = dir !== savedDir.value
  } catch (e) {
    console.error('获取默认目录失败:', e)
  }
}

// --- 默认漫画目录 ---
const defaultComicDir = ref('')
const editingDir = ref(false)
const tempDir = ref('')
const importDirStatus = ref('') // '' | 'loading' | 'done' | 'error'
const importDirMsg = ref('')

const loadDefaultComicDir = async () => {
  try {
    defaultComicDir.value = await GetDefaultComicDir() || ''
  } catch (e) {
    console.error('获取默认漫画目录失败:', e)
  }
}

const startEditDir = () => {
  tempDir.value = defaultComicDir.value
  editingDir.value = true
  nextTick(() => {
    const input = document.querySelector('.dir-input')
    if (input) input.focus()
  })
}

const saveDefaultComicDir = async () => {
  const dir = tempDir.value.trim()
  if (!dir) return
  try {
    await SetDefaultComicDir(dir)
    defaultComicDir.value = dir
    editingDir.value = false
  } catch (e) {
    console.error('保存默认漫画目录失败:', e)
  }
}

const cancelEditDir = () => {
  editingDir.value = false
}

const importDefaultDir = async () => {
  if (!defaultComicDir.value || importDirStatus.value === 'loading') return
  importDirStatus.value = 'loading'
  importDirMsg.value = ''
  try {
    const res = await AddsComic(defaultComicDir.value)
    if (res && res.success) {
      importDirStatus.value = 'done'
      importDirMsg.value = `导入成功: ${res.added} 个，跳过: ${res.skipped} 个`
    } else {
      importDirStatus.value = 'error'
      importDirMsg.value = res?.error || '导入失败'
    }
  } catch (e) {
    importDirStatus.value = 'error'
    importDirMsg.value = e.message
  }
  setTimeout(() => {
    importDirStatus.value = ''
    importDirMsg.value = ''
  }, 3000)
}

const pickDefaultDir = async () => {
  try {
    const dir = await SelectFolder()
    if (dir) {
      tempDir.value = dir
    }
  } catch (e) {
    // 用户取消
  }
}

onMounted(() => {
  fetchTags()
  loadDefaultComicDir()
})
</script>

<template>
  <div class="setting-page">
    <h1 class="setting-page-title">设置</h1>

    <!-- 漫画管理 -->
    <div class="setting-section">
      <div class="section-header">
        <h2 class="section-title">漫画管理</h2>
      </div>

      <!-- 默认漫画目录 -->
      <div class="dir-row">
        <span class="dir-label">默认路径：</span>
        <template v-if="editingDir">
          <input v-model="tempDir" class="dir-input" type="text" placeholder="输入漫画文件夹路径"
            @keyup.enter="saveDefaultComicDir" @keyup.esc="cancelEditDir" />
          <button class="dir-btn" @click="pickDefaultDir">选择</button>
          <button class="dir-btn dir-btn-save" @click="saveDefaultComicDir">保存</button>
          <button class="dir-btn" @click="cancelEditDir">取消</button>
        </template>
        <template v-else>
          <span class="dir-path" :class="{ 'dir-path-empty': !defaultComicDir }" @click="startEditDir">
            {{ defaultComicDir || '未设置（点击选择）' }}
          </span>
          <button v-if="defaultComicDir" class="dir-btn dir-btn-import" :disabled="importDirStatus === 'loading'"
            @click="importDefaultDir">
            {{ importDirStatus === 'loading' ? '导入中...' : '导入' }}
          </button>
          <span v-if="importDirMsg" class="dir-import-msg" :class="'dir-import-' + importDirStatus">
            {{ importDirMsg }}
          </span>
        </template>
      </div>

      <div class="setting-actions">
        <button class="setting-btn" @click="pickMultiple">添加漫画文件夹</button>
      </div>

      <div v-if="paths.length" class="path-list">
        <div class="path-list-header">
          <span class="path-list-title">已选择路径</span>
          <div class="path-list-btns">
            <button class="path-btn path-btn-all" :disabled="loading" @click="confirmAll">全部导入</button>
            <button class="path-btn path-btn-all" @click="cancelAll">全部取消</button>
          </div>
        </div>

        <div v-for="(item, index) in paths" :key="index" class="path-item">
          <span class="path-text">{{ item.path }}</span>
          <div class="path-item-btns">
            <template v-if="item.status === 'pending'">
              <button class="path-btn path-btn-confirm" :disabled="loading" @click="confirmOne(index)">确定导入</button>
              <button class="path-btn path-btn-cancel" @click="cancelOne(index)">取消</button>
            </template>
            <span v-else-if="item.status === 'loading'" class="path-status">导入中...</span>
            <span v-else-if="item.status === 'done'" class="path-status path-status-ok">
              <template v-if="item.result.added !== undefined">
                导入成功: {{ item.result.added }} 个，已存在跳过：{{ item.result.skipped }} 个
              </template>
              <template v-else>
                导入成功:《{{ item.result.title }}》 ({{ item.result.total_pages }}页)
              </template>
            </span>
            <span v-else-if="item.status === 'error'" class="path-status path-status-err">
              {{ item.error }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 标签管理 -->
    <div class="setting-section">
      <div class="section-header tag-section-header">
        <h2 class="section-title">标签管理</h2>
        <button class="setting-btn tag-edit-btn" @click="toggleEditMode">
          {{ editMode ? '完成' : '删除' }}
        </button>
      </div>

      <div v-if="tagsLoading" class="tag-loading">加载中...</div>
      <div v-else-if="tags.length === 0" class="tag-empty">暂无标签</div>
      <div v-else class="tag-list">
        <!-- 添加标签 -->
        <input v-if="showTagInput" v-model="newTag" class="tag-add-input" type="text" placeholder="新标签"
          @keyup.enter="addTag" @blur="addTag" @keyup.esc="showTagInput = false; newTag = ''" />
        <button v-else class="tag-item " @click="showTagInput = true">+</button>

        <div v-for="tag in tags" :key="tag.id" class="tag-item" :class="{ 'tag-item-editing': editMode }">
          <!-- 删除按钮（编辑模式下显示在右上角） -->
          <button v-if="editMode" class="tag-delete-btn" @click.stop="requestDelete(tag.id)">×</button>

          <!-- 标签名（可点击编辑） -->
          <div class="tag-name-wrap">
            <input v-if="editingTagId === tag.id" v-model="editingName" class="tag-name-input"
              @keydown.enter="confirmEditName(tag)" @keydown.escape="cancelEditName" @blur="cancelEditName" />
            <span v-else class="tag-name" @click="startEditName(tag)">{{ tag.name }}</span>
          </div>

          <!-- 关联数量 -->
          <span class="tag-count">{{ tag.bookCount }} 本</span>
        </div>
      </div>
    </div>

    <!-- 下载管理 -->
    <!-- <div class="setting-section">
      <div class="section-header tag-section-header">
        <h2 class="section-title">下载管理</h2>
      </div>
      <div class="download-row">
        <span class="download-label">保存目录：</span>
        <span class="download-path">{{ downloadDir }}</span>
      </div>
      <div class="download-actions">
        <button class="setting-btn" @click="changeDownloadDir">更改目录</button>
        <button class="setting-btn" @click="saveDownloadDir" :disabled="!downloadDirChanged">确定</button>
        <button class="setting-btn" @click="resetDownloadDir">恢复默认</button>
      </div>
    </div> -->

    <!-- 删除确认弹窗 -->
    <div v-if="confirmDeleteId !== null" class="dialog-overlay" @click.self="cancelDelete">
      <div class="dialog-box">
        <p class="dialog-text">确认删除该标签？关联的漫画不会被删除。</p>
        <div class="dialog-btns">
          <button class="dialog-btn dialog-btn-cancel" @click="cancelDelete">取消</button>
          <button class="dialog-btn dialog-btn-confirm" @click="confirmDelete">确认删除</button>
        </div>
      </div>
    </div>

    <button class="setting-exit" @click="goBack">退出</button>
  </div>
</template>

<style scoped>
.setting-page {
  padding: 32px;
  margin: 0 auto;
  position: relative;
  min-height: calc(100vh - 64px);
  box-sizing: border-box;
}

.setting-page-title {
  margin: 0 0 24px;
  font-size: 28px;
  font-weight: 700;
  color: var(--app-text);
}

.setting-section {
  margin-bottom: 24px;
  border: 1px solid var(--sidebar-border);
  border-radius: 8px;
  padding: 16px;
}

.section-title {
  text-align: left;
  margin: 0 0 12px;
  font-size: 18px;
  font-weight: 600;
  color: var(--app-text);
}

.setting-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.setting-btn {
  padding: 8px 20px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  cursor: pointer;
  transition: border-color 0.12s ease, background-color 0.12s ease;
  outline: none;
}

.setting-btn:hover:not(:disabled) {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.setting-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.path-list {
  margin-top: 20px;
  border: 1px solid var(--sidebar-border);
  border-radius: 8px;
  overflow: hidden;
}

.path-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: var(--sidebar-bg);
  border-bottom: 1px solid var(--sidebar-border);
}

.path-list-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.path-list-btns {
  display: flex;
  gap: 8px;
}

.path-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid var(--sidebar-border);
  gap: 12px;
}

.path-item:last-child {
  border-bottom: none;
}

.path-text {
  flex: 1;
  font-size: 13px;
  color: var(--app-text);
  word-break: break-all;
  line-height: 1.5;
  text-align: left;
}

.path-item-btns {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.path-btn {
  padding: 4px 12px;
  border: 1px solid var(--switch-border);
  border-radius: 4px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 0.12s ease, background-color 0.12s ease;
  outline: none;
  white-space: nowrap;
}

.path-btn:hover:not(:disabled) {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.path-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.path-btn-confirm {
  border-color: var(--app-text);
}

.path-status {
  font-size: 12px;
  color: var(--muted-text);
  white-space: nowrap;
}

.path-status-ok {
  color: #2ecc71;
}

.path-status-err {
  color: #e74c3c;
}

.setting-exit {
  position: absolute;
  bottom: 32px;
  right: 32px;
  padding: 8px 20px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  cursor: pointer;
  transition: border-color 0.12s ease, background-color 0.12s ease;
  outline: none;
}

.setting-exit:hover {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.section-header {
  background: var(--sidebar-bg);
  padding: 8px 12px;
  border-radius: 8px 8px 0 0;
  margin: -16px -16px 16px;
}

.tag-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tag-section-header .section-title {
  margin-bottom: 0;
}

.tag-edit-btn {
  padding: 2px 10px;
  font-size: 12px;
  margin-left: 10px;
}

.tag-loading,
.tag-empty {
  color: var(--muted-text);
  font-size: 14px;
  padding: 16px 0;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.tag-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
}

.tag-item-editing {
  padding-right: 14px;
}

.tag-delete-btn {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 50%;
  background: #e74c3c;
  color: #fff;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  padding: 0;
}

.tag-delete-btn:hover {
  background: #c0392b;
}

.tag-add-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 4px 14px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  flex-shrink: 0;
  height: 32px;
  box-sizing: border-box;
}

.tag-add-btn:hover {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.tag-add-input {
  display: inline-block;
  padding: 2px 4px;
  border: 1px solid var(--switch-border);
  border-radius: 3px;
  background: var(--switch-bg);
  color: var(--app-text);
  font-size: 14px;
  outline: none;
  width: 120px;
  height: auto;
  box-sizing: border-box;
}

.tag-add-input:focus {
  border-color: var(--focus-ring);
}

.tag-name-wrap {
  display: flex;
  align-items: center;
}

.tag-name {
  font-size: 14px;
  color: var(--app-text);
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 3px;
}

.tag-name:hover {
  background: var(--switch-bg);
}

.tag-name-input {
  font-size: 14px;
  color: var(--app-text);
  background: var(--switch-bg);
  border: 1px solid var(--switch-border);
  border-radius: 3px;
  padding: 2px 4px;
  outline: none;
  width: 120px;
}

.tag-name-input:focus {
  border-color: var(--focus-ring);
}

.tag-count {
  font-size: 12px;
  color: var(--muted-text);
  white-space: nowrap;
}

/* ===== 删除确认弹窗 ===== */

.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
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
  padding: 24px 28px;
  min-width: 280px;
  max-width: 90vw;
}

.dialog-text {
  font-size: 14px;
  color: var(--app-text);
  margin: 0 0 20px;
  text-align: center;
}

.dialog-btns {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.dialog-btn {
  padding: 6px 20px;
  border: 1px solid var(--switch-border);
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  outline: none;
}

.dialog-btn-cancel {
  background: var(--sidebar-bg);
  color: var(--app-text);
}

.dialog-btn-cancel:hover {
  background: var(--switch-bg);
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

/* ===== 下载管理 ===== */

.download-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.download-label {
  font-size: 14px;
  color: var(--app-text);
  white-space: nowrap;
}

.download-path {
  flex: 1;
  font-size: 13px;
  color: var(--muted-text);
  word-break: break-all;
  line-height: 1.5;
  text-align: left;
}

.download-actions {
  display: flex;
  gap: 8px;
}

/* 默认漫画目录 */
.dir-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.dir-label {
  font-size: 14px;
  color: var(--app-text);
  white-space: nowrap;
}

.dir-path {
  flex: 1;
  font-size: 13px;
  color: var(--muted-text);
  word-break: break-all;
  line-height: 1.5;
  text-align: left;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background-color 0.12s ease;
}

.dir-path:hover {
  background: var(--sidebar-bg);
}

.dir-path-empty {
  color: var(--muted-text);
  opacity: 0.6;
}

.dir-input {
  flex: 1;
  padding: 6px 10px;
  border: 1px solid var(--switch-border);
  border-radius: 4px;
  background: var(--sidebar-bg);
  color: var(--app-text);
  font-size: 13px;
  outline: none;
  min-width: 200px;
}

.dir-input:focus {
  border-color: var(--app-text);
}

.dir-btn {
  padding: 4px 12px;
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

.dir-btn:hover {
  border-color: var(--app-text);
  background: var(--switch-bg);
}

.dir-btn-save {
  border-color: var(--app-text);
}

.dir-btn-import {
  border-color: var(--app-text);
  margin-left: auto;
}

.dir-import-msg {
  font-size: 12px;
  white-space: nowrap;
}

.dir-import-done {
  color: #2ecc71;
}

.dir-import-error {
  color: #e74c3c;
}
</style>
