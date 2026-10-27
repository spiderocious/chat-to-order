import { z } from 'zod'

export const ErrorCode = z.enum([
  'validation', 'malformed_json', 'unauthenticated', 'invalid_credentials', 'token_revoked',
  'not_found', 'unknown_reference', 'conflict', 'invalid_state_transition', 'version_conflict', 'business_rule',
  'idempotency_mismatch', 'rate_limited', 'upstream_unavailable', 'paystack_unavailable', 'store_unavailable', 'internal',
  // client-only
  'network', 'timeout', 'contract_drift',
])
export type ErrorCode = z.infer<typeof ErrorCode>

const Envelope = z.object({
  error: z.object({
    code: z.string(),
    message: z.string(),
    severity: z.string().optional(),
    field_errors: z.array(z.object({ field: z.string(), message: z.string() })).optional(),
    request_id: z.string().optional(),
    current: z.unknown().optional(),
    retry_after: z.number().optional(),
  }),
})

export class ApiError extends Error {
  code: ErrorCode
  status: number
  fieldErrors: { field: string; message: string }[]
  retryAfter: number | undefined
  requestId: string | undefined
  current: unknown

  constructor(code: ErrorCode, message: string, status: number, extra: Partial<Pick<ApiError, 'fieldErrors' | 'retryAfter' | 'requestId' | 'current'>> = {}) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.fieldErrors = extra.fieldErrors ?? []
    this.retryAfter = extra.retryAfter
    this.requestId = extra.requestId
    this.current = extra.current
  }

  static from(res: Response, body: unknown): ApiError {
    const parsed = Envelope.safeParse(body)
    if (!parsed.success) {
      return new ApiError('internal', `Request failed (${res.status}).`, res.status)
    }
    const e = parsed.data.error
    const codeParsed = ErrorCode.safeParse(e.code)
    const code: ErrorCode = codeParsed.success ? codeParsed.data : 'internal'
    const retryHeader = res.headers.get('Retry-After')
    return new ApiError(code, e.message, res.status, {
      fieldErrors: e.field_errors ?? [],
      retryAfter: e.retry_after ?? (retryHeader ? Number(retryHeader) : undefined),
      requestId: e.request_id,
      current: e.current,
    })
  }

  get retryable(): boolean {
    return ['network', 'timeout', 'upstream_unavailable', 'paystack_unavailable', 'store_unavailable', 'internal'].includes(this.code) || this.status >= 500
  }
}

export function isApiError(e: unknown): e is ApiError {
  return e instanceof ApiError
}

export function errorMessage(e: unknown): string {
  if (isApiError(e)) return e.message
  if (e instanceof Error) return e.message
  return 'Something went wrong.'
}
