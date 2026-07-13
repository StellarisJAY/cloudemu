<script setup lang="ts">
import { computed } from 'vue'
import type { PlayerRole } from '@/types/api'
import type { PlayMember } from '@/stores/room'
import { fileUrl } from '@/utils/url'

const props = defineProps<{
  member: PlayMember
  isHost: boolean
  members: PlayMember[]
  maxPorts: number
}>()

const emit = defineEmits<{
  roleChange: [userId: string, role: PlayerRole, port?: number]
  kick: [userId: string]
}>()

const roleLabels: Record<PlayerRole, string> = { 0: '房主', 1: '玩家', 2: '旁观' }
const roleColors: Record<PlayerRole, string> = { 0: '#4fc3f7', 1: '#4ade80', 2: '#f59e0b' }

// 构建端口占用表：port → 占用者名称
const portOccupants = computed(() => {
  const map: Record<number, string> = {}
  for (const m of props.members) {
    if (m.port !== null) {
      map[m.port] = m.nickname || m.username
    }
  }
  return map
})

// 根据成员当前角色动态生成下拉选项
const roleOptions = computed<{ label: string; value: number }[]>(() => {
  const options: { label: string; value: number }[] = []
  for (let p = 0; p < props.maxPorts; p++) {
    const occupant = portOccupants.value[p]
    if (occupant) {
      options.push({ label: `端口${p} [${occupant}]`, value: p })
    } else {
      options.push({ label: `端口${p}`, value: p })
    }
  }
  // 玩家额外可降为旁观
  if (props.member.role === 1) {
    options.push({ label: '降为旁观', value: -1 })
  }
  return options
})

function handleSelect(value: number) {
  if (value === -1) {
    emit('roleChange', props.member.userId, 2)
  } else {
    emit('roleChange', props.member.userId, 1, value)
  }
}

function displayName(m: PlayMember): string {
  return m.nickname || m.username
}
</script>

<template>
  <div class="member-item">
    <div class="member-row member-row--top">
      <!-- 有头像路径走 src + fallback slot（加载失败由 n-avatar 内部自动切到 fallback）；无路径直接 default slot 显示首字母 -->
      <n-avatar v-if="member.avatar" :size="32" :src="fileUrl(member.avatar)" round>
        <template #fallback>
          {{ member.username.charAt(0).toUpperCase() }}
        </template>
      </n-avatar>
      <n-avatar v-else :size="32" round>
        {{ member.username.charAt(0).toUpperCase() }}
      </n-avatar>

      <div class="member-info">
        <span class="member-name">
          {{ displayName(member) }}
          <span v-if="member.isSelf" class="member-self-tag">我</span>
        </span>
        <span class="member-username">@{{ member.username }}</span>
      </div>

      <div class="member-tag">
        <span class="role-dot" :style="{ background: roleColors[member.role] }" />
        <span class="role-label" :style="{ color: roleColors[member.role] }">
          {{ roleLabels[member.role] }}
        </span>
      </div>
    </div>

    <!-- 房主可操作：独立一行 -->
    <div v-if="isHost && !member.isSelf && member.role !== 0" class="member-row member-row--ops">
      <n-select
        :value="member.port ?? null"
        :options="roleOptions"
        size="tiny"
        class="role-select"
        @update:value="handleSelect"
      />
      <n-popconfirm @positive-click="$emit('kick', member.userId)">
        <template #trigger>
          <n-button size="tiny" type="error" quaternary class="kick-btn"> 踢出 </n-button>
        </template>
        确定踢出 {{ displayName(member) }}？
      </n-popconfirm>
    </div>
  </div>
</template>

<style scoped>
.member-item {
  padding: 8px 10px;
  border-radius: var(--radius-md);
  transition: background 0.15s;
}

.member-item:hover {
  background: var(--color-bg-hover);
}

.member-row {
  display: flex;
  align-items: center;
}

.member-row--top {
  gap: 8px;
}

.member-row--ops {
  gap: 6px;
  margin-top: 6px;
  padding-left: 40px; /* 与上方文字对齐 */
}

.member-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.member-name {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-self-tag {
  display: inline-block;
  padding: 0 5px;
  font-size: 10px;
  color: var(--color-accent);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-sm);
  margin-left: 4px;
  vertical-align: middle;
}

.member-username {
  font-size: var(--font-size-mini);
  color: var(--color-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-tag {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.role-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.role-label {
  font-size: 11px;
  font-weight: 500;
}

.role-select {
  width: 130px;
}

.kick-btn {
  font-size: 11px;
}
</style>
