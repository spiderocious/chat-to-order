import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import { Toaster } from 'sonner'
import { routeTree } from './routeTree.gen'
import { AuthProvider } from '@features/auth/providers/auth-provider'
import { isApiError } from '@shared/api/errors'
import './styles/app.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
      retry: (n, err) => n < 2 && (isApiError(err) ? err.retryable : true),
    },
  },
})

const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: 'intent',
  defaultPendingMs: 150,
  defaultPendingMinMs: 300,
  scrollRestoration: true,
})

declare module '@tanstack/react-router' {
  interface Register { router: typeof router }
}

// A chunk that failed to load after a deploy: reload once.
window.addEventListener('vite:preloadError', () => {
  const k = 'ctpo.reloaded'
  if (!sessionStorage.getItem(k)) { sessionStorage.setItem(k, '1'); window.location.reload() }
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RouterProvider router={router} />
        <Toaster position="top-center" richColors={false} toastOptions={{ className: 'font-text !rounded-[8px] !border-2 !border-ink !shadow-[3px_4px_0_#1b2430]' }} />
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>,
)
