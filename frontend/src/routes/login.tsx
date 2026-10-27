import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect, useState, type FormEvent } from 'react'
import { z } from 'zod'
import { motion } from 'motion/react'
import { useAuth } from '@features/auth/providers/auth-provider'
import { Button } from '@shared/ui/Button'
import { Field, Input } from '@shared/ui/primitives'
import { ROUTES } from '@shared/utils/routes'
import { isApiError, errorMessage } from '@shared/api/errors'
import { Eye, EyeOff } from '@icons'

const Search = z.object({ next: z.string().regex(/^\/[^/].*/).optional() })

export const Route = createFileRoute('/login')({ validateSearch: Search, component: Login })

function Login() {
  const { status, login } = useAuth()
  const navigate = useNavigate()
  const { next } = Route.useSearch()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [show, setShow] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [retryIn, setRetryIn] = useState(0)
  const [busy, setBusy] = useState(false)
  const [shake, setShake] = useState(false)

  useEffect(() => { if (status === 'authed') void navigate({ to: next ?? ROUTES.BOARD, replace: true }) }, [status, next, navigate])
  useEffect(() => {
    if (retryIn <= 0) return
    const t = setInterval(() => setRetryIn((n) => n - 1), 1000)
    return () => clearInterval(t)
  }, [retryIn])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await login(email.trim(), password)
    } catch (err) {
      setError(errorMessage(err))
      if (isApiError(err) && err.code === 'rate_limited') setRetryIn(err.retryAfter ?? 60)
      setShake(true)
      setTimeout(() => setShake(false), 300)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="grid min-h-dvh place-items-center bg-paper p-4">
      <motion.form onSubmit={submit} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.28, ease: [0.22, 0.61, 0.36, 1] }}
        className={`pass w-full max-w-sm p-6 ${shake ? 'shake' : ''}`} aria-describedby={error ? 'login-error' : undefined}>
        <div className="mb-5 flex items-center gap-3">
          <span className="grid size-11 place-items-center rounded-[12px] bg-leaf font-display text-[18px] font-extrabold text-white">IB</span>
          <div><h1 className="text-[22px] font-extrabold leading-tight">Iya Bisi's Kitchen</h1><div className="text-[12px] font-extrabold uppercase tracking-[0.1em] text-ink-3">Orders</div></div>
        </div>
        <div className="space-y-3">
          <Field label="Email" htmlFor="email">
            <Input id="email" type="email" autoComplete="username" required value={email} onChange={(e) => setEmail(e.target.value)} error={!!error} />
          </Field>
          <Field label="Password" htmlFor="password">
            <div className="relative">
              <Input id="password" type={show ? 'text' : 'password'} autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} error={!!error} className="pr-10" />
              <button type="button" className="absolute right-2 top-1/2 -translate-y-1/2 grid size-7 place-items-center text-ink-3" onClick={() => setShow((s) => !s)} aria-label={show ? 'Hide password' : 'Show password'}>
                {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
              </button>
            </div>
          </Field>
          {error ? <p id="login-error" role="alert" className="rounded-[6px] bg-critical-soft px-3 py-2 text-[12.5px] font-bold text-critical-on">{error}{retryIn > 0 ? ` (${retryIn}s)` : ''}</p> : null}
          <Button type="submit" size="lg" className="w-full" loading={busy} disabled={retryIn > 0}>{retryIn > 0 ? `Wait ${retryIn}s` : 'Sign in'}</Button>
        </div>
      </motion.form>
    </div>
  )
}
