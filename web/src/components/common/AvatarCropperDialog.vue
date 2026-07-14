<script setup lang="ts">
import { ref, watch, nextTick, onBeforeUnmount } from 'vue'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'

const props = defineProps<{
  show: boolean
  /** 待裁剪的原始图片文件 */
  file: File | null
}>()

const emit = defineEmits<{
  close: []
  /** 裁剪完成，回传裁剪后的正方形图片文件 */
  cropped: [file: File]
}>()

const imageRef = ref<HTMLImageElement | null>(null)
const imageSrc = ref('')
const submitting = ref(false)
let cropper: Cropper | null = null

/** 销毁 cropper 实例并释放预览用的 blob URL */
function destroyCropper() {
  if (cropper) {
    cropper.destroy()
    cropper = null
  }
  if (imageSrc.value) {
    URL.revokeObjectURL(imageSrc.value)
    imageSrc.value = ''
  }
}

/** 弹窗打开且拿到文件时，初始化 cropper（1:1 正方形裁剪框） */
watch(
  () => [props.show, props.file] as const,
  async ([show, file]) => {
    if (show && file) {
      destroyCropper()
      imageSrc.value = URL.createObjectURL(file)
      await nextTick()
      if (!imageRef.value) return
      cropper = new Cropper(imageRef.value, {
        aspectRatio: 1,
        viewMode: 1,
        dragMode: 'move',
        autoCropArea: 0.8,
        background: false,
        responsive: true,
      })
    } else if (!show) {
      destroyCropper()
    }
  },
)

onBeforeUnmount(destroyCropper)

async function handleConfirm() {
  if (!cropper || !props.file) return
  submitting.value = true
  try {
    // 输出 512x512 正方形头像，超出原图尺寸时不放大
    const canvas = cropper.getCroppedCanvas({
      width: 512,
      height: 512,
      imageSmoothingEnabled: true,
      imageSmoothingQuality: 'high',
    })
    const blob = await new Promise<Blob | null>((resolve) => {
      canvas.toBlob((b) => resolve(b), 'image/png')
    })
    if (!blob) {
      submitting.value = false
      return
    }
    const baseName = props.file.name.replace(/\.[^.]+$/, '')
    const croppedFile = new File([blob], `${baseName}.png`, { type: 'image/png' })
    emit('cropped', croppedFile)
  } finally {
    submitting.value = false
  }
}

function handleClose() {
  emit('close')
}

/* ── 变换操作 ── */
function rotate(deg: number) {
  cropper?.rotate(deg)
}

function zoom(ratio: number) {
  cropper?.zoom(ratio)
}

function reset() {
  cropper?.reset()
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="裁剪头像"
    style="width: 440px"
    :mask-closable="false"
    @update:show="(v: boolean) => !v && handleClose()"
  >
    <div class="cropper-body">
      <div class="cropper-container">
        <img ref="imageRef" :src="imageSrc" alt="待裁剪图片" class="cropper-image" />
      </div>
      <div class="cropper-toolbar">
        <n-button size="small" quaternary @click="zoom(0.1)">放大</n-button>
        <n-button size="small" quaternary @click="zoom(-0.1)">缩小</n-button>
        <n-button size="small" quaternary @click="rotate(-90)">左转</n-button>
        <n-button size="small" quaternary @click="rotate(90)">右转</n-button>
        <n-button size="small" quaternary @click="reset()">重置</n-button>
      </div>
    </div>

    <template #footer>
      <div class="form-footer">
        <n-button @click="handleClose">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="handleConfirm"> 确认 </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.cropper-body {
  padding-top: 4px;
}

.cropper-container {
  width: 100%;
  height: 320px;
  background: var(--color-bg-primary);
  border-radius: var(--radius-md);
  overflow: hidden;
}

/* cropper.js 需要 img 为 block 且限定最大高度以便正确初始化 */
.cropper-image {
  display: block;
  max-width: 100%;
  max-height: 320px;
}

.cropper-toolbar {
  display: flex;
  justify-content: center;
  gap: 6px;
  margin-top: 12px;
}

.form-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
