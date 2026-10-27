// The only client-side formatting: relative time and a WAT clock label.
// Money and ready-by labels come from the backend as strings.

export function relativeTime(iso: string, now: Date = new Date()): string {
  const diff = (now.getTime() - new Date(iso).getTime()) / 1000
  if (diff < 45) return 'just now'
  if (diff < 90) return '1 min ago'
  if (diff < 3600) return `${Math.round(diff / 60)} min ago`
  if (diff < 7200) return '1 h ago'
  if (diff < 86400) return `${Math.round(diff / 3600)} h ago`
  return `${Math.round(diff / 86400)} d ago`
}

const watFmt = new Intl.DateTimeFormat('en-GB', { timeZone: 'Africa/Lagos', hour: '2-digit', minute: '2-digit', hour12: false })

export function wat(iso: string | null | undefined): string {
  if (!iso) return ''
  return watFmt.format(new Date(iso))
}

export function titleCase(s: string): string {
  return s.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}
