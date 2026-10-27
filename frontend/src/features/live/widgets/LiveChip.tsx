import { cn } from '@shared/utils/cn'
import { useLive } from '../providers/live-provider'

const copy = {
  live: { dot: 'bg-leaf breathe', text: 'Live', cls: 'bg-success-soft text-success-on border-success-border' },
  degraded: { dot: 'bg-caution', text: 'Updates paused, refreshing every 30 s', cls: 'bg-caution-soft text-caution-on border-caution-border' },
  connecting: { dot: 'bg-ink-4 animate-pulse', text: 'Connecting', cls: 'bg-neutral-soft text-neutral-on border-neutral-border' },
  offline: { dot: 'bg-critical', text: 'Offline, showing last known', cls: 'bg-critical-soft text-critical-on border-critical-border' },
} as const

export function LiveChip({ compact = false }: { compact?: boolean }) {
  const { state, reconnect } = useLive()
  const c = copy[state]
  return (
    <button type="button" onClick={state === 'offline' ? reconnect : undefined} title={c.text} aria-live="polite"
      className={cn('inline-flex h-7 items-center gap-1.5 rounded-full border px-2.5 text-[11px] font-extrabold transition-colors duration-180', c.cls, state !== 'offline' && 'cursor-default')}>
      <span className={cn('size-2 rounded-full', c.dot)} aria-hidden />
      <span className={cn(compact && state !== 'live' && 'sr-only sm:not-sr-only')}>{compact && state === 'live' ? 'Live' : c.text}</span>
    </button>
  )
}
