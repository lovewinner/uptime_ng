import { useAuthStore } from '@/stores/auth'

export class WSClient {
  private ws: WebSocket | null = null
  private reconnectTimer: any = null
  private messageHandler: ((msg: any) => void) | null = null

  connect() {
    const auth = useAuthStore()
    if (!auth.token) return

    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${proto}://${window.location.host}/api/ws?token=${auth.token}`

    this.ws = new WebSocket(url)

    this.ws.onopen = () => console.log('[WS] connected')
    this.ws.onclose = () => {
      console.log('[WS] disconnected, reconnecting in 5s...')
      this.scheduleReconnect()
    }
    this.ws.onerror = (e) => console.error('[WS] error', e)

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (this.messageHandler) {
          this.messageHandler(msg)
        }
      } catch {
        // ignore
      }
    }
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  onMessage(handler: (msg: any) => void) {
    this.messageHandler = handler
  }

  private scheduleReconnect() {
    this.reconnectTimer = setTimeout(() => {
      this.connect()
    }, 5000)
  }
}

export const wsClient = new WSClient()