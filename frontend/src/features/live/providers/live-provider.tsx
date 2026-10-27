import { createContext, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { auth as tokenStore, BASE } from '@shared/api/client'
import { EP } from '@shared/api/endpoints'
import { K } from '@shared/api/keys'

export type LiveState = 'connecting' | 'live' | 'degraded' | 'offline'

interface LiveValue { state: LiveState; lastEventAt: number | null; reconnect: () => void }
const LiveContext = createContext<LiveValue>({ state: 'connecting', lastEventAt: null, reconnect: () => {} })

// SSE over fetch (EventSource cannot send Authorization). Events carry ids and
// types only; every event is a refetch. On every (re)connect: full refetch.
export function LiveProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient()
  const [state, setState] = useState<LiveState>('connecting')
  const [lastEventAt, setLastEventAt] = useState<number | null>(null)
  const attempt = useRef(0)
  const abort = useRef<AbortController | null>(null)
  const degradedServer = useRef(false)
  const [nonce, setNonce] = useState(0)

  useEffect(() => {
    let cancelled = false
    let watchdog: ReturnType<typeof setTimeout> | undefined

    const invalidateFor = (type: string, data: { order_no?: string }) => {
      switch (type) {
        case 'order.created':
        case 'order.updated':
          void qc.invalidateQueries({ queryKey: K.ordersAll })
          void qc.invalidateQueries({ queryKey: K.stats })
          void qc.invalidateQueries({ queryKey: K.activityAll })
          if (data.order_no) void qc.invalidateQueries({ queryKey: K.order(data.order_no) })
          break
        case 'menu.updated':
          void qc.invalidateQueries({ queryKey: K.menu })
          void qc.invalidateQueries({ queryKey: K.stats })
          break
        case 'settings.updated':
          void qc.invalidateQueries({ queryKey: K.vendor })
          void qc.invalidateQueries({ queryKey: K.menu })
          void qc.invalidateQueries({ queryKey: K.stats })
          break
        case 'reset':
          void qc.invalidateQueries()
          break
      }
    }

    const connect = async () => {
      if (cancelled || !tokenStore.token) return
      abort.current?.abort()
      const ctrl = new AbortController()
      abort.current = ctrl
      setState((s) => (s === 'live' ? s : 'connecting'))
      const resetWatchdog = () => {
        if (watchdog) clearTimeout(watchdog)
        watchdog = setTimeout(() => ctrl.abort(), 45_000)
      }
      try {
        const res = await fetch(BASE + EP.EVENTS, { headers: { Authorization: `Bearer ${tokenStore.token}`, Accept: 'text/event-stream' }, signal: ctrl.signal })
        if (!res.ok || !res.body) throw new Error(`sse ${res.status}`)
        attempt.current = 0
        void qc.invalidateQueries()
        resetWatchdog()
        const reader = res.body.getReader()
        const dec = new TextDecoder()
        let buf = ''
        for (;;) {
          const { value, done } = await reader.read()
          if (done) break
          buf += dec.decode(value, { stream: true })
          resetWatchdog()
          let idx: number
          while ((idx = buf.indexOf('\n\n')) >= 0) {
            const frame = buf.slice(0, idx)
            buf = buf.slice(idx + 2)
            let type = 'message'
            let data = '{}'
            for (const line of frame.split('\n')) {
              if (line.startsWith('event:')) type = line.slice(6).trim()
              else if (line.startsWith('data:')) data = line.slice(5).trim()
            }
            if (frame.startsWith(':')) continue
            let payload: { order_no?: string; degraded?: boolean; on?: boolean } = {}
            try { payload = JSON.parse(data) as typeof payload } catch { /* ignore */ }
            if (type === 'hello') {
              degradedServer.current = Boolean(payload.degraded)
              setState(degradedServer.current ? 'degraded' : 'live')
              continue
            }
            if (type === 'degraded') {
              degradedServer.current = Boolean(payload.on)
              setState(degradedServer.current ? 'degraded' : 'live')
              continue
            }
            setLastEventAt(Date.now())
            invalidateFor(type, payload)
          }
        }
        // server closed after its 10 minute cap: reconnect immediately
        if (!cancelled) void connect()
      } catch {
        if (cancelled) return
        attempt.current += 1
        setState(attempt.current >= 3 ? 'offline' : 'connecting')
        const delay = Math.min(30_000, 1000 * 2 ** attempt.current) * (0.8 + Math.random() * 0.4)
        setTimeout(() => { if (!cancelled) void connect() }, delay)
      } finally {
        if (watchdog) clearTimeout(watchdog)
      }
    }

    void connect()
    const onVis = () => { if (document.visibilityState === 'visible') { attempt.current = 0; void connect() } }
    document.addEventListener('visibilitychange', onVis)
    return () => {
      cancelled = true
      document.removeEventListener('visibilitychange', onVis)
      abort.current?.abort()
      if (watchdog) clearTimeout(watchdog)
    }
  }, [qc, nonce])

  const value = useMemo<LiveValue>(() => ({ state, lastEventAt, reconnect: () => { attempt.current = 0; setNonce((n) => n + 1) } }), [state, lastEventAt])
  return <LiveContext.Provider value={value}>{children}</LiveContext.Provider>
}

export function useLive(): LiveValue {
  return useContext(LiveContext)
}

/** Poll interval for list queries: off while live, 30 s otherwise. */
export function usePollInterval(): number | false {
  const { state } = useLive()
  return state === 'live' ? false : 30_000
}
