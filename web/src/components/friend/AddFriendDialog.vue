<script setup lang="ts">
import { ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useFriendStore } from '@/stores/friend'
import { fileUrl } from '@/utils/url'

const friendStore = useFriendStore()
const message = useMessage()

const showModal = ref(false)
const searchQuery = ref('')
const addingId = ref<string | null>(null)
let debounceTimer: ReturnType<typeof setTimeout> | null = null

function open() {
  showModal.value = true
  searchQuery.value = ''
  friendStore.searchResults = []
}

function close() {
  showModal.value = false
}

function onSearchInput(val: string) {
  if (debounceTimer) clearTimeout(debounceTimer)
  if (!val.trim()) {
    friendStore.searchResults = []
    return
  }
  debounceTimer = setTimeout(() => {
    friendStore.searchUsers(val.trim())
  }, 300)
}

watch(searchQuery, onSearchInput)

/** 检查是否已是好友 */
function isFriend(userId: string): boolean {
  return friendStore.friends.some((f) => f.user_id === userId || f.friend_id === userId)
}

/** 检查是否在待处理列表中（我发出的或收到的） */
function isPending(userId: string): boolean {
  return friendStore.pendingList.some((p) => p.user_id === userId)
}

/** 获取搜索结果操作按钮的状态 */
function getButtonState(userId: string): 'add' | 'pending' | 'friend' {
  if (isFriend(userId)) return 'friend'
  if (isPending(userId)) return 'pending'
  return 'add'
}

async function handleAdd(userId: string) {
  addingId.value = userId
  const err = await friendStore.addFriend(userId)
  addingId.value = null
  if (err) {
    message.error(err)
    return
  }
  message.success('好友申请已发送')
}

function displayName(u: { username: string; nickname: string | null }): string {
  return u.nickname || u.username
}
</script>

<template>
  <div class="add-friend-trigger">
    <n-button size="small" type="primary" secondary @click="open"> 添加好友 </n-button>

    <n-modal
      v-model:show="showModal"
      preset="card"
      title="添加好友"
      :mask-closable="true"
      style="max-width: 400px"
      class="add-friend-modal"
    >
      <div class="modal-content">
        <n-input
          v-model:value="searchQuery"
          placeholder="搜索用户名..."
          size="medium"
          clearable
          class="search-input"
        />

        <!-- 搜索中 -->
        <div v-if="friendStore.searchLoading" class="search-status">搜索中...</div>

        <!-- 搜索结果 -->
        <div v-else-if="friendStore.searchResults.length > 0" class="search-results">
          <div v-for="user in friendStore.searchResults" :key="user.id" class="search-item">
            <n-avatar v-if="user.avatar" :size="32" :src="fileUrl(user.avatar)" round>
              <template #fallback>
                {{ user.username.charAt(0).toUpperCase() }}
              </template>
            </n-avatar>
            <n-avatar v-else :size="32" round>
              {{ user.username.charAt(0).toUpperCase() }}
            </n-avatar>
            <div class="search-item-info">
              <span class="search-item-name">{{ displayName(user) }}</span>
              <span class="search-item-username">@{{ user.username }}</span>
            </div>
            <div class="search-item-action">
              <n-button
                v-if="getButtonState(user.id) === 'add'"
                size="tiny"
                type="primary"
                secondary
                :loading="addingId === user.id"
                @click="handleAdd(user.id)"
              >
                添加
              </n-button>
              <n-tag v-else-if="getButtonState(user.id) === 'pending'" size="small" type="warning">
                待接受
              </n-tag>
              <n-tag v-else size="small" type="info"> 已添加 </n-tag>
            </div>
          </div>
        </div>

        <!-- 无结果 / 未搜索 -->
        <div v-else-if="searchQuery.trim() && !friendStore.searchLoading" class="search-status">
          未找到匹配的用户
        </div>
        <div v-else class="search-status">输入用户名开始搜索</div>
      </div>

      <template #footer>
        <n-button text type="primary" size="small" @click="close"> 关闭 </n-button>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.modal-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 200px;
}

.search-input {
  flex-shrink: 0;
}

.search-results {
  display: flex;
  flex-direction: column;
  max-height: 240px;
  overflow-y: auto;
}

.search-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 4px;
  border-bottom: 1px solid var(--color-divider);
  transition: background 0.15s;
}

.search-item:last-child {
  border-bottom: none;
}

.search-item:hover {
  background: var(--color-bg-hover);
  margin: 0 -4px;
  padding: 8px 8px;
  border-radius: var(--radius-sm);
}

.search-item-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.search-item-name {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.search-item-username {
  font-size: var(--font-size-mini);
  color: var(--color-text-secondary);
}

.search-item-action {
  flex-shrink: 0;
}

.search-status {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
  padding: 24px 0;
}

/* 结果列表滚动条 */
.search-results::-webkit-scrollbar {
  width: 4px;
}

.search-results::-webkit-scrollbar-thumb {
  background: var(--color-scrollbar);
  border-radius: 2px;
}
</style>
