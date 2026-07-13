import { ref, onUnmounted } from 'vue'
import { Room, RoomEvent, Track, type RemoteTrack } from 'livekit-client'
import type { ConnectionState } from '@/components/play/GameToolbar.vue'

export function useLiveKit() {
  const room = ref<Room | null>(null)
  const videoTrack = ref<MediaStreamTrack | null>(null)
  const audioTrack = ref<MediaStreamTrack | null>(null)
  const connectionState = ref<ConnectionState>('waiting')
  const error = ref<string | null>(null)

  let dataReceivedCallback: ((payload: Uint8Array, topic: string) => void) | null = null

  function setOnDataReceived(cb: typeof dataReceivedCallback) {
    dataReceivedCallback = cb
  }

  async function connect(url: string, token: string) {
    if (room.value) await disconnect()
    connectionState.value = 'connecting'
    error.value = null

    try {
      const r = new Room()
      room.value = r
      console.log("livkit url: ", url)
      await r.prepareConnection(url, token)

      r.on(RoomEvent.TrackSubscribed, (track: RemoteTrack) => {
        if (track.kind === Track.Kind.Video) {
          videoTrack.value = track.mediaStreamTrack ?? null
        } else if (track.kind === Track.Kind.Audio) {
          audioTrack.value = track.mediaStreamTrack ?? null
        }
      })

      r.on(RoomEvent.Connected, () => {
        connectionState.value = 'connected'
      })

      r.on(RoomEvent.DataReceived, (payload: Uint8Array, _participant, _kind, topic?: string) => {
        if (dataReceivedCallback) {
          const copy = new Uint8Array(payload)
          dataReceivedCallback(copy, topic ?? '')
        }
      })

      r.on(RoomEvent.Disconnected, () => {
        connectionState.value = 'waiting'
        room.value = null
        videoTrack.value = null
        audioTrack.value = null
      })

      await r.connect(url, token)
    } catch (e: unknown) {
      connectionState.value = 'error'
      error.value = e instanceof Error ? e.message : '连接失败'
    }
  }

  async function disconnect() {
    if (room.value) {
      await room.value.disconnect()
      room.value = null
    }
    videoTrack.value = null
    audioTrack.value = null
    connectionState.value = 'waiting'
  }

  function publishInput(data: Uint8Array, topic?: string) {
    const opts: { reliable: boolean; topic?: string } = { reliable: false }
    if (topic) opts.topic = topic
    room.value?.localParticipant.publishData(data, opts)
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    room,
    videoTrack,
    audioTrack,
    connectionState,
    error,
    connect,
    disconnect,
    publishInput,
    setOnDataReceived,
  }
}
