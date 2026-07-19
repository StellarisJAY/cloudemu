<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useRoomStore } from '@/stores/room'
import { useFriendStore } from '@/stores/friend'
import { useAuthStore } from '@/stores/auth'
import type { EmulatorType, FriendWithUser } from '@/types/api'

const props = defineProps<{
  show: boolean
  prefillTitle?: string
  prefillEmulatorType?: EmulatorType | null
  prefillRomId?: string | null
}>()

const emit = defineEmits<{
  close: []
  created: [roomId: string]
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

const emulatorLocked = computed(() => props.prefillEmulatorType != null)

// 弹窗打开时确保用户信息和好友列表已加载，并应用预填值
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
    // 应用预填值
    title.value = props.prefillTitle || ''
    if (props.prefillEmulatorType) {
      emulatorType.value = props.prefillEmulatorType
    }
  },
)

const emulatorOptions = [
  { label: 'NES', value: 'nes' as EmulatorType },
  { label: 'GBC/GBA', value: 'gb' as EmulatorType },
  { label: 'DOS', value: 'dos' as EmulatorType },
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
  const result = await roomStore.createRoom({
    title: title.value.trim(),
    emulator_type: emulatorType.value,
    max_ports: maxPorts.value,
    rom_id: props.prefillRomId || undefined,
    invitee_ids: selectedFriendIds.value.length > 0 ? selectedFriendIds.value : undefined,
  })
  submitting.value = false

  if (typeof result === 'string') {
    message.error(result)
    return
  }

  message.success('房间创建成功')
  emit('created', result.id)
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
        <n-select
          v-model:value="emulatorType"
          :options="emulatorOptions"
          :disabled="emulatorLocked"
        />
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
