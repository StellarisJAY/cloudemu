<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useMediaQuery } from '@vueuse/core'
import { useAuthStore } from '@/stores/auth'
import { useRoomStore } from '@/stores/room'
import { useRomStore } from '@/stores/rom'
import FriendList from '@/components/friend/FriendList.vue'
import RoomList from '@/components/room/RoomList.vue'
import CreateRoomDialog from '@/components/room/CreateRoomDialog.vue'
import RomList from '@/components/rom/RomList.vue'
import RomUploadDialog from '@/components/rom/RomUploadDialog.vue'
import RomEditDialog from '@/components/rom/RomEditDialog.vue'
import MobileLobby from '@/components/lobby/MobileLobby.vue'
import type { Rom } from '@/types/api'

// 竖屏移动端：房间/ROM/好友需拆分为独立页面，用 MobileLobby 承载
const isMobile = useMediaQuery('(pointer: coarse) and (max-width: 768px)')

const auth = useAuthStore()
const roomStore = useRoomStore()
const romStore = useRomStore()
const router = useRouter()
const message = useMessage()

const showCreateRoom = ref(false)
const showUploadRom = ref(false)
const showEditRom = ref(false)
const editingRom = ref<Rom | null>(null)

const showDeleteConfirm = ref(false)
const pendingDeleteRoomId = ref<string | null>(null)

const deletingRoom = computed(() => roomStore.rooms.find((r) => r.id === pendingDeleteRoomId.value))

function handleRoomDelete(roomId: string) {
  pendingDeleteRoomId.value = roomId
  showDeleteConfirm.value = true
}

async function confirmDelete() {
  if (!pendingDeleteRoomId.value) return
  const err = await roomStore.deleteRoom(pendingDeleteRoomId.value)
  if (err) {
    message.error(err)
  } else {
    message.success('房间已删除')
    await roomStore.fetchRooms()
  }
  showDeleteConfirm.value = false
  pendingDeleteRoomId.value = null
}

onMounted(() => {
  if (!auth.user) {
    auth.fetchUser()
  }
  roomStore.fetchRooms()
  romStore.fetchRoms()
})

function handleRoomClick(roomId: string) {
  router.push(`/play/${roomId}`)
}

function handleRomClick(romId: string) {
  const rom = romStore.roms.find((r) => r.id === romId)
  if (rom) {
    editingRom.value = rom
    showEditRom.value = true
  }
}

function handleRomEditClose() {
  showEditRom.value = false
  editingRom.value = null
}

function handleRomEdited() {
  showEditRom.value = false
  editingRom.value = null
}
</script>

<template>
  <!-- 移动端竖屏：底部 Tab 切换房间 / ROM / 好友 -->
  <MobileLobby
    v-if="isMobile"
    :rooms="roomStore.rooms"
    :room-loading="roomStore.loading"
    :roms="romStore.roms"
    :rom-loading="romStore.loading"
    :current-user-id="auth.user?.id"
    @create="showCreateRoom = true"
    @room-click="handleRoomClick"
    @room-delete="handleRoomDelete"
    @upload="showUploadRom = true"
    @rom-click="handleRomClick"
  />

  <!-- 桌面端：三栏布局 -->
  <div v-else class="lobby-layout">
    <!-- 顶栏 -->
    <header class="lobby-header">
      <div class="header-left">
        <h1 class="header-logo">CloudEmu</h1>
      </div>
      <div class="header-right">
        <span v-if="auth.user" class="header-user">
          {{ auth.user.nickname || auth.user.username }}
        </span>
        <n-button size="small" text @click="router.push('/profile')"> 设置 </n-button>
        <n-button size="small" text @click="auth.logout"> 退出 </n-button>
      </div>
    </header>

    <!-- 主体 -->
    <div class="lobby-body">
      <!-- 左侧好友列表 -->
      <aside class="lobby-sidebar">
        <FriendList />
      </aside>

      <!-- 中央内容区：上下两部分 -->
      <main class="lobby-main">
        <!-- 上半部分：房间列表 -->
        <div class="lobby-section lobby-section--top">
          <RoomList
            :rooms="roomStore.rooms"
            :loading="roomStore.loading"
            :current-user-id="auth.user?.id"
            @create="showCreateRoom = true"
            @room-click="handleRoomClick"
            @room-delete="handleRoomDelete"
          />
        </div>

        <!-- 分隔线 -->
        <div class="section-divider" />

        <!-- 下半部分：ROM 库 -->
        <div class="lobby-section lobby-section--bottom">
          <RomList
            :roms="romStore.roms"
            :loading="romStore.loading"
            @upload="showUploadRom = true"
            @rom-click="handleRomClick"
          />
        </div>
      </main>
    </div>
  </div>

  <!-- 弹窗（桌面/移动端共用） -->
  <CreateRoomDialog
      :show="showCreateRoom"
      @close="showCreateRoom = false"
      @created="showCreateRoom = false"
    />
    <RomUploadDialog
      :show="showUploadRom"
      @close="showUploadRom = false"
      @uploaded="showUploadRom = false"
    />
    <RomEditDialog
      :show="showEditRom"
      :rom="editingRom"
      @close="handleRomEditClose"
      @updated="handleRomEdited"
    />

    <!-- 删除房间确认 -->
    <n-modal
      v-model:show="showDeleteConfirm"
      preset="dialog"
      title="删除房间"
      positive-text="确认删除"
      negative-text="取消"
      type="warning"
      @positive-click="confirmDelete"
      @negative-click="()=>{
        showDeleteConfirm = false
        pendingDeleteRoomId = null
      }
      "
    >
      <p>
        确定要删除房间「<strong>{{ deletingRoom?.title }}</strong
        >」吗？
      </p>
      <p style="color: var(--color-text-secondary); font-size: var(--font-size-small)">
        所有房间成员和关联数据都将被永久删除，且无法恢复。
      </p>
    </n-modal>
</template>

<style scoped>
.lobby-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--color-bg-primary);
}

/* ── 顶栏 ── */
.lobby-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 16px;
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
}

.header-logo {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--color-accent);
  letter-spacing: 1px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-user {
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
}

/* ── 主体 ── */
.lobby-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.lobby-sidebar {
  width: 260px;
  flex-shrink: 0;
  overflow: hidden;
}

.lobby-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 20px 0 0;
}

.lobby-section {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.lobby-section--top {
  flex: 1;
}

.lobby-section--bottom {
  flex: 1;
}

.section-divider {
  height: 1px;
  margin: 16px 16px;
  background: var(--color-divider);
  flex-shrink: 0;
}
</style>
