<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useRoomStore } from '@/stores/room'
import { useFriendStore } from '@/stores/friend'
import { useAuthStore } from '@/stores/auth'
import type { EmulatorType, FriendWithUser } from '@/types/api'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  created: []
}>()

const message = useMessage()
const roomStore = useRoomStore()
const friendStore = useFriendStore()
const authStore = useAuthStore()

const title = ref('')
const emulatorType = ref<EmulatorType>('nes')
const maxPorts = ref(4)
const selectedFriendIds = ref<string[]>([])
const submitting = ref(false)

// 弹窗打开时确保用户信息和好友列表已加载
watch(
  () => props.show,
  async (val) => {
    if (!val) return
    if (!authStore.user) {
      await authStore.fetchUser()
    }
    if (friendStore.friends.length === 0) {
      await friendStore.fetchFriends()
    }
  },
)

const emulatorOptions = [
  { label: 'NES', value: 'nes' as const },
  { label: 'GBC/GBA', value: 'gba' as const },
  { label: 'DOS', value: 'dos' as const },
]

const portOptions = [
  { label: '1 人', value: 1 },
  { label: '2 人', value: 2 },
  { label: '3 人', value: 3 },
  { label: '4 人', value: 4 },
]

function displayName(f: { username: string; nickname: string | null }): string {
  return f.nickname || f.username
}

/**
 * 获取好友的实际用户ID
 * friend_id 和 user_id 中有一个是当前用户，返回另一个（即好友的真实用户ID）
 */
function friendUserId(f: FriendWithUser): string {
  const currentId = authStore.user?.id
  if (currentId && f.user_id === currentId) return f.friend_id
  return f.user_id
}

const friendOptions = computed(() =>
  friendStore.friends.map((f: FriendWithUser) => ({
    label: `${displayName(f)} (@${f.username})`,
    value: friendUserId(f),
  })),
)

async function handleSubmit() {
  if (!title.value.trim()) {
    message.warning('请输入房间名称')
    return
  }

  submitting.value = true
  const err = await roomStore.createRoom({
    title: title.value.trim(),
    emulator_type: emulatorType.value,
    max_ports: maxPorts.value,
    invitee_ids: selectedFriendIds.value.length > 0 ? selectedFriendIds.value : undefined,
  })
  submitting.value = false

  if (err) {
    message.error(err)
    return
  }

  message.success('房间创建成功')
  emit('created')
  resetForm()
}

function handleClose() {
  resetForm()
  emit('close')
}

function resetForm() {
  title.value = ''
  emulatorType.value = 'nes'
  maxPorts.value = 4
  selectedFriendIds.value = []
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="创建房间"
    style="width: 480px"
    :mask-closable="false"
    @update:show="(v: boolean) => !v && handleClose()"
  >
    <n-form label-placement="top" class="create-room-form">
      <n-form-item label="房间名称" required>
        <n-input v-model:value="title" placeholder="输入房间名称" maxlength="128" />
      </n-form-item>

      <n-form-item label="模拟器类型" required>
        <n-select v-model:value="emulatorType" :options="emulatorOptions" />
      </n-form-item>

      <n-form-item label="最大玩家数" required>
        <n-radio-group v-model:value="maxPorts">
          <n-radio v-for="opt in portOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </n-radio>
        </n-radio-group>
      </n-form-item>

      <n-form-item label="邀请好友（可选）">
        <n-select
          v-model:value="selectedFriendIds"
          :options="friendOptions"
          multiple
          placeholder="选择要邀请的好友，好友直接加入无需接受"
          filterable
          clearable
        />
      </n-form-item>
    </n-form>

    <template #footer>
      <div class="form-footer">
        <n-button @click="handleClose">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="handleSubmit"> 创建 </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.create-room-form {
  padding-top: 8px;
}

.form-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
