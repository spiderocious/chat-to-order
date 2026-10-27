import { createRootRouteWithContext, Outlet } from '@tanstack/react-router'
import type { QueryClient } from '@tanstack/react-query'
import { ErrorState } from '@shared/ui/primitives'
import { Link } from '@tanstack/react-router'
import { ROUTES } from '@shared/utils/routes'

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: () => <Outlet />,
  errorComponent: ({ error, reset }) => (
    <div className="grid min-h-dvh place-items-center p-4">
      <div className="pass max-w-md p-6">
        <ErrorState error={error} onRetry={reset} />
        <div className="text-center"><Link to={ROUTES.BOARD} className="text-[13px] font-bold text-leaf-700 underline">Back to board</Link></div>
      </div>
    </div>
  ),
  notFoundComponent: () => (
    <div className="grid min-h-dvh place-items-center p-4">
      <div className="pass max-w-md p-6 text-center">
        <h1 className="text-[20px] font-extrabold">Nothing here</h1>
        <p className="mt-1 text-ink-3">That page does not exist.</p>
        <Link to={ROUTES.BOARD} className="mt-3 inline-block text-[13px] font-bold text-leaf-700 underline">Back to board</Link>
      </div>
    </div>
  ),
})
