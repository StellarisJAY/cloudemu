<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useFriendStore } from '@/stores/friend'
import FriendList from '@/components/friend/FriendList.vue'
import RoomList from '@/components/room/RoomList.vue'
import RomList from '@/components/rom/RomList.vue'
import type { Room, Rom } from '@/types/api'

defineProps<{
  rooms: Room[]
  roomLoading: boolean
  roms: Rom[]
  romLoading: boolean
  currentUserId?: string
}>()

defineEmits<{
  create: []
  'room-click': [roomId: string]
  'room-delete': [roomId: string]
  'room-stop': [roomId: string]
  'room-leave': [roomId: string]
  upload: []
  'rom-click': [romId: string]
}>()

type Tab = 'rooms' | 'roms' | 'friends'

const auth = useAuthStore()
const friendStore = useFriendStore()
const router = useRouter()

const activeTab = ref<Tab>('rooms')
</script>

<template>
  <div class="mobile-lobby">
    <!-- 顶栏 -->
    <header class="mobile-header">
      <h1 class="header-logo">CloudEmu</h1>
      <div class="header-right">
        <span v-if="auth.user" class="header-user">
          {{ auth.user.nickname || auth.user.username }}
        </span>
        <n-button size="tiny" text @click="router.push('/profile')">设置</n-button>
        <n-button size="tiny" text @click="auth.logout">退出</n-button>
      </div>
    </header>

    <!-- 内容区：每个 Tab 独占一屏，v-show 保持组件挂载与滚动位置 -->
    <main class="mobile-body">
      <section v-show="activeTab === 'rooms'" class="mobile-pane">
        <RoomList
          :rooms="rooms"
          :loading="roomLoading"
          :current-user-id="currentUserId"
          class="mobile-list"
          @create="$emit('create')"
          @room-click="(id: string) => $emit('room-click', id)"
          @room-delete="(id: string) => $emit('room-delete', id)"
          @room-stop="(id: string) => $emit('room-stop', id)"
          @room-leave="(id: string) => $emit('room-leave', id)"
        />
      </section>

      <section v-show="activeTab === 'roms'" class="mobile-pane">
        <RomList
          :roms="roms"
          :loading="romLoading"
          class="mobile-list"
          @upload="$emit('upload')"
          @rom-click="(id: string) => $emit('rom-click', id)"
        />
      </section>

      <section v-show="activeTab === 'friends'" class="mobile-pane">
        <FriendList class="mobile-friends" />
      </section>
    </main>

    <!-- 底部 Tab 栏 -->
    <nav class="mobile-tabbar">
      <button
        class="tab-item"
        :class="{ active: activeTab === 'rooms' }"
        @click="activeTab = 'rooms'"
      >
        <span class="tab-icon">🎮</span>
        <span class="tab-label">房间</span>
      </button>
      <button
        class="tab-item"
        :class="{ active: activeTab === 'roms' }"
        @click="activeTab = 'roms'"
      >
        <span class="tab-icon">💾</span>
        <span class="tab-label">ROM 库</span>
      </button>
      <button
        class="tab-item"
        :class="{ active: activeTab === 'friends' }"
        @click="activeTab = 'friends'"
      >
        <span class="tab-icon">
          👥
          <n-badge
            v-if="friendStore.pendingCount > 0"
            :value="friendStore.pendingCount"
            type="error"
            :offset="[6, -2]"
          />
        </span>
        <span class="tab-label">好友</span>
      </button>
    </nav>
  </div>
</template>

<style scoped>
.mobile-lobby {
  display: flex;
  flex-direction: column;
  height: 100vh;
  height: 100dvh;
  background: var(--color-bg-primary);
  overflow: hidden;
}

/* ── 顶栏 ── */
.mobile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  padding: 0 12px;
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.header-logo {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-accent);
  letter-spacing: 1px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-user {
  font-size: var(--font-size-mini);
  color: var(--color-text-secondary);
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ── 内容区 ── */
.mobile-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.mobile-pane {
  height: 100%;
  overflow-y: auto;
  padding-top: 14px;
}

/* 房间 / ROM 列表：竖屏改为纵向自适应网格，不再横向滚动 */
.mobile-list {
  height: 100%;
}

.mobile-list :deep(.card-grid) {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  overflow-x: visible;
  align-content: start;
}

.mobile-list :deep(.card-grid > *) {
  width: 100% !important;
}

.mobile-friends {
  height: 100%;
}

/* 好友面板在竖屏铺满，无需右边框 */
.mobile-friends :deep(.friend-panel) {
  border-right: none;
}

/* ── 底部 Tab 栏 ── */
.mobile-tabbar {
  display: flex;
  flex-shrink: 0;
  height: 56px;
  padding-bottom: env(safe-area-inset-bottom, 0);
  background: var(--color-bg-secondary);
  border-top: 1px solid var(--color-border);
}

.tab-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border: none;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  touch-action: manipulation;
  transition: color 0.15s;
}

.tab-item.active {
  color: var(--color-accent);
}

.tab-item:active {
  background: var(--color-bg-hover);
}

.tab-icon {
  position: relative;
  font-size: 20px;
  line-height: 1;
}

.tab-label {
  font-size: var(--font-size-mini);
}

.mobile-pane::-webkit-scrollbar {
  width: 4px;
}

.mobile-pane::-webkit-scrollbar-thumb {
  background: var(--color-scrollbar);
  border-radius: 2px;
}
</style>
