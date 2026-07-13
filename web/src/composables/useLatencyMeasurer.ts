import { ref, watch, onUnmounted, type Ref } from 'vue'

const PACKET_TYPE_PING = 0x03
const PACKET_TYPE_PONG = 0x04
const PING_INTERVAL = 3000

/**
 * useLatencyMeasurer
 * 通过 LiveKit DataChannel 发送 ping 包到 EmuRunner 并接收 pong 回复，
 * 计算客户端到 EmuRunner 的往返延迟 (RTT)，显示在前端 HUD 上
 *
 * 协议：
 *   ping: [0x03][client_ts:8B int64 LE]    topic="ping", lossy
 *   pong: [0x04][client_ts:8B][server_ts:8B]  topic="ping", lossy, 定向回复
 *
 * @param publishData  LiveKit DataChannel 发送函数
 * @param enabled      是否启用延迟探测（连接成功后为 true）
 * @param onDataReceived 注册 DataChannel 接收回调
 */
export function useLatencyMeasurer(
  publishData: (data: Uint8Array, topic: string) => void,
  enabled: Ref<boolean>,
  onDataReceived: (cb: (payload: Uint8Array, topic: string) => void) => void,
) {
  const latencyMs = ref<number | null>(null)
  let timer: ReturnType<typeof setInterval> | null = null

  function sendPing() {
    const ts = performance.now()
    const buf = new Uint8Array(9)
    buf[0] = PACKET_TYPE_PING
    const view = new DataView(buf.buffer)
    view.setBigInt64(1, BigInt(Math.round(ts)), true) // little-endian
    publishData(buf, 'ping')
  }

  function handlePong(payload: Uint8Array, topic: string) {
    if (topic !== 'ping') return
    if (payload.length < 17 || payload[0] !== PACKET_TYPE_PONG) return
    const view = new DataView(payload.buffer, payload.byteOffset, payload.byteLength)
    const clientTs = Number(view.getBigInt64(1, true))
    latencyMs.value = Math.max(1, Math.round(performance.now() - clientTs))
  }

  onDataReceived(handlePong)

  watch(
    enabled,
    (on) => {
      if (on) {
        sendPing()
        timer = setInterval(sendPing, PING_INTERVAL)
      } else {
        if (timer) {
          clearInterval(timer)
          timer = null
        }
        latencyMs.value = null
      }
    },
    { immediate: true },
  )

  onUnmounted(() => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  })

  return { latencyMs }
}
