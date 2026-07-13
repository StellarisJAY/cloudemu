<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useTheme } from '@/composables/useTheme'

const { isDark } = useTheme()
const canvasRef = ref<HTMLCanvasElement | null>(null)

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  radius: number
  highlight: boolean // 亮节点 → 稍大、发光
}

interface ShootingStar {
  x: number
  y: number
  vx: number
  vy: number
  life: number
  maxLife: number
}

const PARTICLE_COUNT = 90
const MAX_LINE_DIST = 140
const SHOOTING_STAR_INTERVAL = 4000 // ms

let ctx: CanvasRenderingContext2D | null = null
let width = 0
let height = 0
let animId = 0

let particles: Particle[] = []
const shootingStars: ShootingStar[] = []

/* ── 调色板，根据主题切换 ── */
function colors() {
  return isDark.value
    ? { dot: '180,210,255', dotBright: '79,195,247', line: '79,195,247', star: '79,195,247' }
    : { dot: '2,132,199', dotBright: '56,189,248', line: '2,132,199', star: '56,189,248' }
}

function initParticles() {
  particles = Array.from({ length: PARTICLE_COUNT }, () => ({
    x: Math.random() * width,
    y: Math.random() * height,
    vx: (Math.random() - 0.5) * 0.35,
    vy: (Math.random() - 0.5) * 0.35,
    radius: Math.random() * 1.8 + 0.6,
    highlight: Math.random() < 0.12, // 12% 为亮节点
  }))
}

function draw() {
  if (!ctx) return
  const c = colors()

  ctx.clearRect(0, 0, width, height)

  // 1) 粒子连线（星座网状）
  for (let i = 0; i < particles.length; i++) {
    const a = particles[i]!
    for (let j = i + 1; j < particles.length; j++) {
      const b = particles[j]!
      const dx = a.x - b.x
      const dy = a.y - b.y
      const dist = Math.sqrt(dx * dx + dy * dy)

      if (dist < MAX_LINE_DIST) {
        const alpha = 1 - dist / MAX_LINE_DIST
        ctx.strokeStyle = `rgba(${c.line},${(alpha * 0.4).toFixed(2)})`
        ctx.lineWidth = 0.5
        ctx.beginPath()
        ctx.moveTo(a.x, a.y)
        ctx.lineTo(b.x, b.y)
        ctx.stroke()
      }
    }
  }

  // 2) 粒子圆点
  for (const p of particles) {
    if (p.highlight) {
      // 发光节点 — 外光晕 + 内圆
      const grad = ctx.createRadialGradient(p.x, p.y, 0, p.x, p.y, p.radius * 4)
      grad.addColorStop(0, `rgba(${c.dotBright},0.7)`)
      grad.addColorStop(1, `rgba(${c.dotBright},0)`)
      ctx.fillStyle = grad
      ctx.beginPath()
      ctx.arc(p.x, p.y, p.radius * 4, 0, Math.PI * 2)
      ctx.fill()
    }

    ctx.fillStyle = p.highlight ? `rgba(${c.dotBright},0.9)` : `rgba(${c.dot},0.5)`
    ctx.beginPath()
    ctx.arc(p.x, p.y, p.radius, 0, Math.PI * 2)
    ctx.fill()
  }

  // 3) 流星
  for (let i = shootingStars.length - 1; i >= 0; i--) {
    const s = shootingStars[i]!
    s.x += s.vx
    s.y += s.vy
    s.life--
    if (s.life <= 0) {
      shootingStars.splice(i, 1)
      continue
    }
    const progress = s.life / s.maxLife
    const alpha = progress.toFixed(2)

    // 流星头部亮尾
    const grad = ctx.createLinearGradient(s.x, s.y, s.x - s.vx * 8, s.y - s.vy * 8)
    grad.addColorStop(0, `rgba(${c.star},${alpha})`)
    grad.addColorStop(1, `rgba(${c.star},0)`)
    ctx.strokeStyle = grad
    ctx.lineWidth = 1.6
    ctx.lineCap = 'round'
    ctx.beginPath()
    ctx.moveTo(s.x, s.y)
    ctx.lineTo(s.x - s.vx * 8, s.y - s.vy * 8)
    ctx.stroke()
  }
}

function update() {
  for (const p of particles) {
    p.x += p.vx
    p.y += p.vy
    // 边界回弹
    if (p.x < 0 || p.x > width) p.vx *= -1
    if (p.y < 0 || p.y > height) p.vy *= -1
  }
}

function spawnShootingStar() {
  // 从左上/右上区域出现，向左下/右下飞
  const startSide = Math.random() > 0.5 ? 0 : 1 // 0=左上, 1=右上
  const x = startSide === 0 ? Math.random() * width * 0.4 : width - Math.random() * width * 0.4
  const y = Math.random() * height * 0.3
  const angle = Math.PI / 4 + (Math.random() - 0.5) * 0.4 // 约 45° ± 23°
  const speed = 4 + Math.random() * 3
  const maxLife = 60 + Math.floor(Math.random() * 40)

  shootingStars.push({
    x,
    y,
    vx: startSide === 0 ? Math.cos(angle) * speed : -Math.cos(angle) * speed,
    vy: Math.sin(angle) * speed,
    life: maxLife,
    maxLife,
  })
}

function loop() {
  update()
  draw()
  animId = requestAnimationFrame(loop)
}

let shootingTimer: ReturnType<typeof setInterval> | null = null

function onResize() {
  if (!canvasRef.value) return
  width = window.innerWidth
  height = window.innerHeight
  canvasRef.value.width = width
  canvasRef.value.height = height
  initParticles()
}

onMounted(() => {
  ctx = canvasRef.value!.getContext('2d')
  onResize()
  window.addEventListener('resize', onResize)

  shootingTimer = setInterval(spawnShootingStar, SHOOTING_STAR_INTERVAL)
  loop()
})

onBeforeUnmount(() => {
  cancelAnimationFrame(animId)
  window.removeEventListener('resize', onResize)
  if (shootingTimer) clearInterval(shootingTimer)
})

// 主题切换时重置粒子颜色（无需重建粒子）
watch(isDark, () => {
  /* draw() 已动态读取 colors() */
})
</script>

<template>
  <canvas ref="canvasRef" class="animated-bg" />
</template>

<style scoped>
.animated-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
}
</style>
