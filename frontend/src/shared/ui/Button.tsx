import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { cn } from '@shared/utils/cn'
import { Loader2 } from '@icons'

type Variant = 'solid' | 'outline' | 'plain' | 'ghost'
type Size = 'sm' | 'md' | 'lg'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  destructive?: boolean
  size?: Size
  loading?: boolean
  icon?: ReactNode
}

const base = 'inline-flex items-center justify-center gap-2 font-bold rounded-[8px] transition-[transform,background-color,color,border-color] duration-100 ease-[var(--ease-standard)] active:scale-[0.98] disabled:opacity-45 disabled:pointer-events-none select-none whitespace-nowrap'
const sizes: Record<Size, string> = { sm: 'h-8 px-3 text-[12.5px]', md: 'h-10 px-4 text-[13.5px]', lg: 'h-11 px-5 text-[14px]' }

function variantClass(v: Variant, destructive: boolean): string {
  if (destructive) {
    switch (v) {
      case 'solid': return 'bg-danger text-white border-2 border-ink hover:bg-danger-hover'
      case 'outline': return 'border-2 border-danger text-danger bg-card hover:bg-critical-soft'
      case 'plain': return 'text-danger hover:bg-critical-soft'
      case 'ghost': return 'text-danger'
    }
  }
  switch (v) {
    case 'solid': return 'bg-leaf text-white border-2 border-ink hover:bg-leaf-600 shadow-[2px_3px_0_#1b2430] active:shadow-none active:translate-x-[2px] active:translate-y-[3px]'
    case 'outline': return 'border-2 border-ink text-ink bg-card hover:bg-paper-2'
    case 'plain': return 'text-ink-2 hover:bg-paper-2'
    case 'ghost': return 'text-leaf-700 hover:underline'
  }
}

export function Button({ variant = 'solid', destructive = false, size = 'md', loading = false, icon, className, children, disabled, ...rest }: ButtonProps) {
  return (
    <button className={cn(base, sizes[size], variantClass(variant, destructive), className)} disabled={disabled || loading} aria-busy={loading} {...rest}>
      {loading ? <Loader2 className="size-4 animate-spin" aria-hidden /> : icon}
      <span>{children}</span>
    </button>
  )
}
