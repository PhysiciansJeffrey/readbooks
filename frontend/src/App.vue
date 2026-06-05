
<script setup>
import { ref, provide, nextTick } from 'vue'
import { RouterView } from 'vue-router'
import Sidebar from '@/components/Sidebar.vue'
import Searchbar from '@/components/Searchbar.vue'
import RandomFloatBtn from '@/components/RandomFloatBtn.vue'

const keepAliveIncludes = ref(['Home', 'Search'])

provide('refreshHome', () => {
  keepAliveIncludes.value = keepAliveIncludes.value.filter(v => v !== 'Home')
  nextTick(() => {
    keepAliveIncludes.value.push('Home')
  })
})

const searchbarRef = ref(null)

const onTopHover = () => {
  searchbarRef.value?.onTopHover?.()
}

const onTopLeave = () => {
  searchbarRef.value?.onTopLeave?.()
}
</script>

<template>
  <!-- 顶部热区：鼠标移入时触发搜索栏显示 -->
  <div
    class="top-hover-zone"
    @mouseenter="onTopHover"
    @mouseleave="onTopLeave"
  ></div>

  <Sidebar/>
  <Searchbar ref="searchbarRef"/>
  <RouterView v-slot="{ Component }">
    <KeepAlive :include="keepAliveIncludes">
      <component :is="Component" />
    </KeepAlive>
  </RouterView>
  <RandomFloatBtn :pages="[]" />
</template>

<style scoped>
/* 顶部不可见热区 */
.top-hover-zone {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60px;
  z-index: 9;
  width: 70%;
}
</style>
