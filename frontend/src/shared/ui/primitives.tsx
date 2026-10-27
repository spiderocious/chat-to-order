import { useEffect, useRef, useState, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes } from 'react'
import { cn } from '@shared/utils/cn'
import { Inbox, CircleAlert, RefreshCw, Minus, Plus } from '@icons'
import { Button } from './Button'
import { errorMessage, isApiError } from '@shared/api/errors'

// ---------- PASS / BOARD ----------

export function Pass({ className, children }: { className?: string; children: ReactNode }) {
  return <section className={cn('pass p-4 sm:p-5', className)}>{children}</section>
}

export function Board({ className, children }: { className?: string; children: ReactNode }) {
  return <section className={cn('board overflow-hidden', className)}>{children}</section>
}

export function BoardHead({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn('hairline bg-paper-2 px-3 py-2 text-[10.5px] font-extrabold uppercase tracking-[0.09em] text-ink-3', className)}>{children}</div>
}

export function SectionTitle({ children, aside }: { children: ReactNode; aside?: ReactNode }) {
  return (
    <div className="mb-3 flex items-baseline justify-between gap-3">
      <h2 className="text-[17px] font-extrabold">{children}</h2>
      {aside}
    </div>
  )
}

// ---------- Figures: flip, never tween. A ring pulse draws the eye. ----------

export function Flip({ value, className }: { value: string | number; className?: string }) {
  const [pulse, setPulse] = useState(false)
  const prev = useRef(value)
  useEffect(() => {
    if (prev.current !== value) {
      prev.current = value
      setPulse(true)
      const t = setTimeout(() => setPulse(false), 500)
      return () => clearTimeout(t)
    }
    return undefined
  }, [value])
  return <span className={cn('tnum inline-block rounded-md', pulse && 'ring-pulse', className)}>{value}</span>
}

export function StatTile({ label, value, hint, tone = 'neutral' }: { label: string; value: string | number; hint?: string; tone?: 'neutral' | 'success' | 'caution' | 'critical' }) {
  const color = { neutral: 'text-ink', success: 'text-leaf-700', caution: 'text-caution-on', critical: 'text-critical' }[tone]
  return (
    <div className="pass-xs px-3.5 py-3">
      <div className="text-[10px] font-extrabold uppercase tracking-[0.08em] text-ink-4">{label}</div>
      <div className={cn('font-display text-[26px] leading-none font-extrabold mt-1', color)}>
        <Flip value={value} />
      </div>
      {hint ? <div className="mt-1 text-[11.5px] text-ink-3">{hint}</div> : null}
    </div>
  )
}

export function StatTileSkeleton() {
  return (
    <div className="pass-xs px-3.5 py-3">
      <div className="skeleton h-3 w-16" />
      <div className="skeleton mt-2 h-7 w-20" />
    </div>
  )
}

// ---------- Inputs: disabled / readOnly / error are independent ----------

export function Field({ label, hint, error, children, htmlFor }: { label: string; hint?: string; error?: string; children: ReactNode; htmlFor?: string }) {
  return (
    <label htmlFor={htmlFor} className="block">
      <span className="mb-1 block text-[12px] font-extrabold text-ink-2">{label}</span>
      {children}
      {error ? <span className="mt-1 block text-[12px] font-bold text-critical" role="alert">{error}</span> : hint ? <span className="mt-1 block text-[11.5px] text-ink-3">{hint}</span> : null}
    </label>
  )
}

const inputBase = 'w-full h-10 rounded-[8px] border bg-card px-3 text-[14px] text-ink transition-[border-color,box-shadow] duration-100 placeholder:text-ink-4 focus-visible:border-leaf disabled:bg-paper-2 disabled:text-ink-4 read-only:bg-paper-2'

export function Input({ error, className, ...rest }: InputHTMLAttributes<HTMLInputElement> & { error?: boolean }) {
  return <input className={cn(inputBase, error ? 'border-critical' : 'border-line-2', className)} aria-invalid={error || undefined} {...rest} />
}

export function Select({ error, className, children, ...rest }: SelectHTMLAttributes<HTMLSelectElement> & { error?: boolean }) {
  return (
    <select className={cn(inputBase, 'pr-8', error ? 'border-critical' : 'border-line-2', className)} aria-invalid={error || undefined} {...rest}>
      {children}
    </select>
  )
}

export function Switch({ checked, onChange, disabled, label }: { checked: boolean; onChange: (v: boolean) => void; disabled?: boolean; label: string }) {
  return (
    <button type="button" role="switch" aria-checked={checked} aria-label={label} disabled={disabled} onClick={() => onChange(!checked)}
      className={cn('relative h-6 w-11 shrink-0 rounded-full border-2 border-ink transition-colors duration-180 disabled:opacity-45', checked ? 'bg-leaf' : 'bg-paper-3')}>
      <span className={cn('absolute top-0.5 size-4 rounded-full bg-white border border-ink transition-transform duration-180 ease-[var(--ease-standard)]', checked ? 'translate-x-[22px]' : 'translate-x-0.5')} />
    </button>
  )
}

export function Stepper({ value, onChange, min = 0, max = 10000, disabled, label }: { value: number; onChange: (v: number) => void; min?: number; max?: number; disabled?: boolean; label: string }) {
  return (
    <div className="inline-flex items-center rounded-[8px] border border-line-2 bg-card" role="group" aria-label={label}>
      <button type="button" className="h-8 w-8 grid place-items-center text-ink-2 hover:bg-paper-2 disabled:opacity-40 rounded-l-[7px]" disabled={disabled || value <= min} onClick={() => onChange(value - 1)} aria-label="decrease"><Minus className="size-3.5" /></button>
      <span className="tnum w-10 text-center text-[13px] font-extrabold"><Flip value={value} /></span>
      <button type="button" className="h-8 w-8 grid place-items-center text-ink-2 hover:bg-paper-2 disabled:opacity-40 rounded-r-[7px]" disabled={disabled || value >= max} onClick={() => onChange(value + 1)} aria-label="increase"><Plus className="size-3.5" /></button>
    </div>
  )
}

// ---------- States ----------

export function EmptyState({ title, body, action }: { title: string; body?: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-6 py-10 text-center">
      <Inbox className="size-7 text-ink-4" aria-hidden />
      <div className="font-extrabold">{title}</div>
      {body ? <p className="max-w-sm text-[13px] text-ink-3">{body}</p> : null}
      {action}
    </div>
  )
}

export function ErrorState({ error, onRetry, compact = false }: { error: unknown; onRetry?: () => void; compact?: boolean }) {
  const msg = errorMessage(error)
  const retryable = isApiError(error) ? error.retryable : true
  return (
    <div className={cn('flex flex-col items-center justify-center gap-2 text-center', compact ? 'px-4 py-5' : 'px-6 py-10')} role="alert">
      <CircleAlert className="size-6 text-critical" aria-hidden />
      <div className="text-[13.5px] font-bold text-ink-2">{msg}</div>
      {onRetry && retryable ? <Button size="sm" variant="outline" icon={<RefreshCw className="size-3.5" />} onClick={onRetry}>Retry</Button> : null}
    </div>
  )
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn('skeleton', className)} aria-hidden />
}

export function RowSkeletons({ n = 5 }: { n?: number }) {
  return (
    <div>
      {Array.from({ length: n }).map((_, i) => (
        <div key={i} className="hairline flex items-center gap-3 px-3 py-2.5">
          <Skeleton className="h-5 w-16 rounded-full" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-4 flex-1" />
          <Skeleton className="h-4 w-14" />
        </div>
      ))}
    </div>
  )
}
