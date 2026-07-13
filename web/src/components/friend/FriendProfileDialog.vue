<script setup lang="ts">
import { ref } from 'vue'
import { authApi } from '@/api/auth'
import { fileUrl } from '@/utils/url'
import type { UserProfile } from '@/types/api'

const showModal = ref(false)
const loading = ref(false)
const profile = ref<UserProfile | null>(null)

async function open(userId: string) {
  showModal.value = true
  loading.value = true
  profile.value = null
  try {
    const res = await authApi.getUser(userId)
    profile.value = res.data.data ?? null
  } catch {
    profile.value = null
  } finally {
    loading.value = false
  }
}

function close() {
  showModal.value = false
}

function displayName(): string {
  if (!profile.value) return ''
  return profile.value.nickname || profile.value.username
}

defineExpose({ open })
</script>

<template>
  <n-modal
    v-model:show="showModal"
    preset="card"
    title="好友信息"
    :mask-closable="true"
    style="max-width: 400px"
    class="profile-modal"
  >
    <div class="profile-content">
      <n-spin v-if="loading" class="profile-loading" />

      <template v-else-if="profile">
        <div class="profile-header">
          <n-avatar
            v-if="profile.avatar"
            :size="80"
            :src="fileUrl(profile.avatar)"
            round
          >
            <template #fallback>
              {{ profile.username.charAt(0).toUpperCase() }}
            </template>
          </n-avatar>
          <n-avatar v-else :size="80" round>
            {{ profile.username.charAt(0).toUpperCase() }}
          </n-avatar>
        </div>

        <div class="profile-name">
          {{ displayName() }}
        </div>
        <div class="profile-username">
          @{{ profile.username }}
        </div>

        <div class="profile-divider" />

        <div class="profile-bio-label">个人简介</div>
        <div class="profile-bio">
          {{ profile.bio || '暂无简介' }}
        </div>
      </template>

      <div v-else class="profile-error">
        加载失败，请稍后重试
      </div>
    </div>

    <template #footer>
      <n-button text type="primary" size="small" @click="close">
        关闭
      </n-button>
    </template>
  </n-modal>
</template>

<style scoped>
.profile-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 200px;
  padding: 8px 0;
}

.profile-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.profile-header {
  margin-bottom: 12px;
}

.profile-name {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.profile-username {
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.profile-divider {
  width: 60px;
  height: 1px;
  background: var(--color-divider);
  margin: 16px 0;
}

.profile-bio-label {
  font-size: var(--font-size-mini);
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
}

.profile-bio {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  text-align: center;
  max-width: 260px;
  line-height: 1.6;
  word-break: break-word;
}

.profile-error {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
}
</style>
