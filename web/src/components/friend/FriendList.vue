<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { useFriendStore } from '@/stores/friend'
import { useAuthStore } from '@/stores/auth'
import { fileUrl } from '@/utils/url'
import AddFriendDialog from './AddFriendDialog.vue'
import FriendProfileDialog from './FriendProfileDialog.vue'
import type { FriendWithUser } from '@/types/api'

const friendStore = useFriendStore()
const authStore = useAuthStore()
const message = useMessage()
const profileDialog = ref<InstanceType<typeof FriendProfileDialog> | null>(null)

onMounted(() => {
  friendStore.fetchFriends()
  friendStore.fetchPending()
})

async function handleAccept(friendId: string) {
  const err = await friendStore.acceptFriend(friendId)
  if (err) message.error(err)
  else message.success('已接受好友申请')
}

async function handleReject(friendId: string) {
  const err = await friendStore.rejectFriend(friendId)
  if (err) message.error(err)
}

/** 获取好友的显示名（昵称优先，否则用户名） */
function displayName(f: { username: string; nickname: string | null }): string {
  return f.nickname || f.username
}

/** 从 FriendWithUser 中提取好友的用户 ID */
function friendUserId(f: FriendWithUser): string {
  return f.user_id === authStore.user?.id ? f.friend_id : f.user_id
}

function handleFriendClick(f: FriendWithUser) {
  profileDialog.value?.open(friendUserId(f))
}
</script>

<template>
  <div class="friend-panel">
    <div class="friend-panel-header">
      <h3 class="friend-panel-title">
        好友列表
        <n-badge
          v-if="friendStore.pendingCount > 0"
          :value="friendStore.pendingCount"
          type="error"
          class="pending-badge"
        />
      </h3>
      <AddFriendDialog />
    </div>

    <div class="friend-panel-body">
      <!-- 加载中 -->
      <div v-if="friendStore.loading" class="friend-empty">加载中...</div>

      <!-- 待处理请求 -->
      <div v-if="friendStore.pendingList.length > 0" class="pending-section">
        <div class="section-label">待处理请求</div>
        <div
          v-for="item in friendStore.pendingList"
          :key="item.id"
          class="friend-item pending-item"
        >
          <n-avatar
            v-if="item.avatar"
            :size="32"
            :src="fileUrl(item.avatar)"
            round
            class="friend-avatar"
          >
            <template #fallback>
              {{ item.username.charAt(0).toUpperCase() }}
            </template>
          </n-avatar>
          <n-avatar v-else :size="32" round class="friend-avatar">
            {{ item.username.charAt(0).toUpperCase() }}
          </n-avatar>
          <div class="friend-info">
            <span class="friend-name">{{ displayName(item) }}</span>
            <span class="friend-username">@{{ item.username }}</span>
          </div>
          <div class="friend-actions">
            <n-button size="tiny" type="success" secondary @click="handleAccept(item.user_id)">
              接受
            </n-button>
            <n-button size="tiny" type="error" secondary @click="handleReject(item.user_id)">
              拒绝
            </n-button>
          </div>
        </div>
      </div>

      <!-- 好友列表 -->
      <div v-if="friendStore.friends.length > 0" class="friends-section">
        <div v-if="friendStore.pendingList.length > 0" class="section-label">
          已添加好友（{{ friendStore.friends.length }}）
        </div>
        <div
          v-for="f in friendStore.friends"
          :key="f.id"
          class="friend-item friend-clickable"
          @click="handleFriendClick(f)"
        >
          <n-avatar
            v-if="f.avatar"
            :size="32"
            :src="fileUrl(f.avatar)"
            round
            class="friend-avatar"
          >
            <template #fallback>
              {{ f.username.charAt(0).toUpperCase() }}
            </template>
          </n-avatar>
          <n-avatar v-else :size="32" round class="friend-avatar">
            {{ f.username.charAt(0).toUpperCase() }}
          </n-avatar>
          <div class="friend-info">
            <span class="friend-name">{{ displayName(f) }}</span>
            <span class="friend-username">@{{ f.username }}</span>
          </div>
          <div class="friend-status">
            <span class="status-dot" />
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div
        v-if="
          !friendStore.loading &&
          friendStore.friends.length === 0 &&
          friendStore.pendingList.length === 0
        "
        class="friend-empty"
      >
        暂无好友，点击上方按钮添加
      </div>
    </div>

    <FriendProfileDialog ref="profileDialog" />
  </div>
</template>

<style scoped>
.friend-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
  overflow: hidden;
}

.friend-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--color-divider);
  flex-shrink: 0;
}

.friend-panel-title {
  margin: 0;
  font-size: var(--font-size-small);
  font-weight: 600;
  color: var(--color-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.pending-badge {
  margin-left: 4px;
}

.friend-panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 6px 0;
}

.section-label {
  padding: 6px 14px 2px;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.friend-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  transition: background 0.15s;
  cursor: default;
}

.friend-item:hover {
  background: var(--color-bg-hover);
}

.friend-clickable {
  cursor: pointer;
}

.pending-item {
  background: var(--color-bg-tertiary);
}

.friend-avatar {
  flex-shrink: 0;
}

.friend-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.friend-name {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.friend-username {
  font-size: var(--font-size-mini);
  color: var(--color-text-secondary);
}

.friend-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.friend-status {
  flex-shrink: 0;
}

.status-dot {
  display: block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-text-tertiary);
}

.friend-empty {
  padding: 24px 14px;
  text-align: center;
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
}

/* 滚动条 */
.friend-panel-body::-webkit-scrollbar {
  width: 4px;
}

.friend-panel-body::-webkit-scrollbar-thumb {
  background: var(--color-scrollbar);
  border-radius: 2px;
}

.friend-panel-body::-webkit-scrollbar-thumb:hover {
  background: var(--color-scrollbar-hover);
}
</style>
