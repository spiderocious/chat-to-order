import type { ReactNode } from 'react'
import { cn } from '@shared/utils/cn'
import type { OrderStatus } from '@shared/api/types'

export type Tone = 'neutral' | 'info' | 'success' | 'caution' | 'critical'

const tones: Record<Tone, string> = {
  neutral: 'bg-neutral-soft text-neutral-on border-neutral-border',
  info: 'bg-info-soft text-info-on border-info-border',
  success: 'bg-success-soft text-success-on border-success-border',
  caution: 'bg-caution-soft text-caution-on border-caution-border',
  critical: 'bg-critical-soft text-critical-on border-critical-border',
}

export function Chip({ tone = 'neutral', children, className, outline = false }: { tone?: Tone; children: ReactNode; className?: string; outline?: boolean }) {
  return (
    <span className={cn('inline-flex items-center gap-1 h-6 px-2.5 rounded-full border text-[11px] font-extrabold uppercase tracking-[0.05em] whitespace-nowrap transition-colors duration-180', tones[tone], outline && 'bg-card', className)}>
      {children}
    </span>
  )
}

export const statusTone: Record<OrderStatus, Tone> = {
  pending_payment: 'caution',
  paid: 'success',
  paid_needs_attention: 'critical',
  resolved: 'success',
  cancelled: 'neutral',
  expired: 'neutral',
}

export const statusLabel: Record<OrderStatus, string> = {
  pending_payment: 'Awaiting pay',
  paid: 'Paid',
  paid_needs_attention: 'Needs you',
  resolved: 'Resolved',
  cancelled: 'Cancelled',
  expired: 'Expired',
}

export function StatusChip({ status, className }: { status: OrderStatus; className?: string }) {
  return (
    <Chip tone={statusTone[status]} outline={status === 'resolved'} className={className}>
      {statusLabel[status]}
    </Chip>
  )
}
