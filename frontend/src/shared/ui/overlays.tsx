import { useEffect, useId, useState, type ReactNode } from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { cn } from '@shared/utils/cn'
import { X } from '@icons'
import { Button } from './Button'
import { Input, Field } from './primitives'
import { useMedia } from '@shared/hooks/use-media'

// One overlay contract: open / onOpenChange / Content. Surfaces differ only in
// anchor and coverage. Scrim 0→.5 in 200 ms; content trails by 40 ms.

interface OverlayProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  children: ReactNode
  footer?: ReactNode
  description?: string
}

function useEscape(open: boolean, close: () => void) {
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') close() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, close])
}

function Scrim({ onClick }: { onClick: () => void }) {
  return <motion.div className="fixed inset-0 z-[400] bg-ink/50" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} transition={{ duration: 0.2 }} onClick={onClick} aria-hidden />
}

export function Modal({ open, onOpenChange, title, description, children, footer }: OverlayProps) {
  const id = useId()
  useEscape(open, () => onOpenChange(false))
  return (
    <AnimatePresence>
      {open ? (
        <>
          <Scrim onClick={() => onOpenChange(false)} />
          <div className="fixed inset-0 z-[500] grid place-items-center p-4 pointer-events-none">
            <motion.div role="dialog" aria-modal="true" aria-labelledby={id} className="pointer-events-auto w-full max-w-md bg-card border-2 border-ink rounded-[12px_4px_12px_4px] shadow-[5px_6px_0_#1b2430] p-5"
              initial={{ opacity: 0, scale: 0.96, y: 8 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.96, y: 8 }} transition={{ duration: 0.28, delay: 0.04, ease: [0.22, 0.61, 0.36, 1] }}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 id={id} className="text-[18px] font-extrabold">{title}</h2>
                  {description ? <p className="mt-1 text-[13px] text-ink-3">{description}</p> : null}
                </div>
                <button className="grid size-8 place-items-center rounded-md text-ink-3 hover:bg-paper-2" onClick={() => onOpenChange(false)} aria-label="Close"><X className="size-4" /></button>
              </div>
              <div className="mt-4">{children}</div>
              {footer ? <div className="mt-5 flex justify-end gap-2">{footer}</div> : null}
            </motion.div>
          </div>
        </>
      ) : null}
    </AnimatePresence>
  )
}

/** Drawer from the right on wide screens, bottom sheet on phones. */
export function Drawer({ open, onOpenChange, title, description, children, footer }: OverlayProps) {
  const id = useId()
  const phone = useMedia('(max-width: 639px)')
  useEscape(open, () => onOpenChange(false))
  return (
    <AnimatePresence>
      {open ? (
        <>
          <Scrim onClick={() => onOpenChange(false)} />
          <motion.aside role="dialog" aria-modal="true" aria-labelledby={id}
            className={cn('fixed z-[500] bg-card border-ink flex flex-col', phone ? 'inset-x-0 bottom-0 max-h-[92dvh] rounded-t-[16px] border-2 border-b-0' : 'top-0 right-0 h-full w-[440px] max-w-[92vw] border-l-2 shadow-[-5px_0_0_#1b2430]')}
            initial={phone ? { y: '100%' } : { x: '100%' }} animate={phone ? { y: 0 } : { x: 0 }} exit={phone ? { y: '100%' } : { x: '100%' }}
            transition={{ duration: 0.28, delay: 0.04, ease: [0.22, 0.61, 0.36, 1] }}
            drag={phone ? 'y' : false} dragConstraints={{ top: 0, bottom: 0 }} dragElastic={{ top: 0, bottom: 0.4 }}
            onDragEnd={(_, info) => { if (phone && info.offset.y > 120) onOpenChange(false) }}>
            {phone ? <div className="mx-auto mt-2 h-1.5 w-10 rounded-full bg-line-2" aria-hidden /> : null}
            <div className="flex items-start justify-between gap-3 px-5 pt-4 pb-3 hairline">
              <div>
                <h2 id={id} className="text-[18px] font-extrabold">{title}</h2>
                {description ? <p className="mt-0.5 text-[13px] text-ink-3">{description}</p> : null}
              </div>
              <button className="grid size-8 place-items-center rounded-md text-ink-3 hover:bg-paper-2" onClick={() => onOpenChange(false)} aria-label="Close"><X className="size-4" /></button>
            </div>
            <div className="flex-1 overflow-y-auto px-5 py-4">{children}</div>
            {footer ? <div className="hairline border-t flex justify-end gap-2 px-5 py-3 safe-bottom">{footer}</div> : null}
          </motion.aside>
        </>
      ) : null}
    </AnimatePresence>
  )
}

/** Typed confirm for the irreversible: the button stays disabled until the word matches. */
export function ConfirmTyped({ open, onOpenChange, title, description, word, confirmLabel, onConfirm, loading, children }: {
  open: boolean; onOpenChange: (o: boolean) => void; title: string; description: string; word: string; confirmLabel: string; onConfirm: () => void; loading?: boolean; children?: ReactNode
}) {
  const [typed, setTyped] = useState('')
  useEffect(() => { if (!open) setTyped('') }, [open])
  return (
    <Modal open={open} onOpenChange={onOpenChange} title={title} description={description}
      footer={<>
        <Button variant="outline" onClick={() => onOpenChange(false)}>Keep it</Button>
        <Button destructive disabled={typed !== word} loading={loading} onClick={onConfirm}>{confirmLabel}</Button>
      </>}>
      {children}
      <Field label={`Type ${word} to confirm`}>
        <Input autoFocus value={typed} onChange={(e) => setTyped(e.target.value.toUpperCase())} placeholder={word} className="mono" />
      </Field>
    </Modal>
  )
}
