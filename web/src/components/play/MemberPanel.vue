<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import { useRoomStore } from '@/stores/room'
import type { PlayMember } from '@/stores/room'
import { useFriendStore } from '@/stores/friend'
import { useAuthStore } from '@/stores/auth'
import MemberItem from './MemberItem.vue'
import type { PlayerRole } from '@/types/api'
import type { FriendWithUser } from '@/types/api'

const props = defineProps<{
  members: PlayMember[]
  roomId: string
  isHost: boolean
  maxPorts: number
}>()

const emit = defineEmits<{
  roleChange: [userId: string, role: PlayerRole, port?: number]
  kick: [userId: string]
  invited: []
}>()

const message = useMessage()
const roomStore = useRoomStore()
const friendStore = useFriendStore()
const authStore = useAuthStore()

const showInvite = ref(false)
const selectedFriendIds = ref<string[]>([])

function showInviteDialog() {
  showInvite.value = true
}

function displayName(f: { username: string; nickname: string | null }): string {
  return f.nickname || f.username
}

function friendUserId(f: FriendWithUser): string {
  const currentId = authStore.user?.id
  if (currentId && f.user_id === currentId) return f.friend_id
  return f.user_id
}

const friendOptions = computed(() =>
  friendStore.friends.map((f: FriendWithUser) => ({
    label: displayName(f) + ' (@' + f.username + ')',
    value: friendUserId(f),
  })),
)

async function handleInvite() {
  if (selectedFriendIds.value.length === 0) {
    message.warning('请选择要邀请的好友')
    return
  }
  const err = await roomStore.inviteToRoom({
    room_id: props.roomId,
    invitee_ids: selectedFriendIds.value,
  })
  if (err) {
    message.error(err)
    return
  }
  message.success('好友已加入房间')
  selectedFriendIds.value = []
  showInvite.value = false
  emit('invited')
}
</script>

<template>
  <div class="member-panel">
    <!-- 头部 -->
    <div class="panel-header">
      <span class="panel-title">
        房间成员
        <span class="panel-count">({{ members.length }})</span>
      </span>
    </div>

    <!-- 成员列表 -->
    <div class="panel-body">
      <div v-if="members.length === 0" class="panel-empty">暂无成员</div>
      <MemberItem
        v-for="m in members"
        :key="m.userId"
        :member="m"
        :is-host="props.isHost"
        :members="props.members"
        :max-ports="props.maxPorts"
        @role-change="(userId, role, port) => emit('roleChange', userId, role, port)"
        @kick="(userId) => emit('kick', userId)"
      />
    </div>

    <!-- 底部邀请 -->
    <div class="panel-footer">
      <n-button block secondary @click="showInviteDialog"> 邀请好友 </n-button>
    </div>

    <!-- 邀请弹窗 -->
    <n-modal v-model:show="showInvite" preset="card" title="邀请好友加入房间" style="width: 400px">
      <n-select
        v-model:value="selectedFriendIds"
        :options="friendOptions"
        multiple
        placeholder="选择要邀请的好友"
        filterable
        class="invite-select"
      />
      <template #footer>
        <div class="modal-footer">
          <n-button @click="showInvite = false">取消</n-button>
          <n-button type="primary" @click="handleInvite">邀请</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.member-panel {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-secondary);
  border-left: 1px solid var(--color-border);
}

.panel-header {
  padding: 16px;
  border-bottom: 1px solid var(--color-divider);
  flex-shrink: 0;
}

.panel-title {
  font-size: var(--font-size-small);
  font-weight: 600;
  color: var(--color-text-primary);
}

.panel-count {
  font-weight: 400;
  color: var(--color-text-secondary);
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 6px 8px;
}

.panel-body::-webkit-scrollbar {
  width: 4px;
}

.panel-body::-webkit-scrollbar-thumb {
  background: var(--color-scrollbar);
  border-radius: 2px;
}

.panel-body::-webkit-scrollbar-thumb:hover {
  background: var(--color-scrollbar-hover);
}

.panel-empty {
  padding: 24px;
  text-align: center;
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
}

.panel-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--color-divider);
  flex-shrink: 0;
}

/* ── 弹窗 ── */
.invite-select {
  min-height: 120px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
