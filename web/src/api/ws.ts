import { useAuthStore } from '@/stores/auth'
import type { WSMessage } from '@/api/types'

export class WSClient {
  private ws: WebSocket | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private messageHandlers = new Set<(msg: WSMessage) => void>()
  private manuallyClosed = false

  connect() {
    const auth = useAuthStore()
    if (!auth.token) return
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) return
    this.manuallyClosed = false

    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${proto}://${window.location.host}/api/ws?token=${encodeURIComponent(auth.token)}`

    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer)
        this.reconnectTimer = null
      }
    }
    this.ws.onclose = () => {
      this.ws = null
      if (!this.manuallyClosed) {
        this.scheduleReconnect()
      }
    }
    this.ws.onerror = () => {
      this.ws?.close()
    }

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data) as WSMessage
        this.messageHandlers.forEach((handler) => handler(msg))
      } catch {
        // ignore
      }
    }
  }

  disconnect() {
    this.manuallyClosed = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  onMessage(handler: (msg: WSMessage) => void): () => void {
    this.messageHandlers.add(handler)
    return () => this.messageHandlers.delete(handler)
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, 5000)
  }
}

export const wsClient = new WSClient()
