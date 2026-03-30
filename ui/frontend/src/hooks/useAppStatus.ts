import useSWR from 'swr'
import type { AuthStatus, GlobalStatus } from '../types'

export interface AppStatus {
  auth: AuthStatus | null
  status: GlobalStatus | null
  initialLoading: boolean
  fetchStatus: () => void
}

const fetcher = (url: string) => fetch(url).then(r => r.json())

/**
 * useAppStatus fetches /api/auth and /api/status and keeps them fresh
 * via swr polling. Auth is fetched once (revalidateOnFocus only).
 * Status refreshes every 10 s; call fetchStatus() to trigger an immediate
 * revalidation after an operation completes.
 *
 * Extracted from AppInner (Item 5.12). Polling upgraded from manual
 * setInterval to swr (Item 5.17).
 */
export function useAppStatus(): AppStatus {
  const { data: auth } = useSWR<AuthStatus>('/api/auth', fetcher, {
    revalidateOnFocus: false,
    revalidateOnReconnect: false,
  })

  const { data: status, mutate: mutateStatus } = useSWR<GlobalStatus>('/api/status', fetcher, {
    refreshInterval: 10000,
    refreshWhenHidden: false,
  })

  const initialLoading = auth === undefined && status === undefined

  const fetchStatus = () => { mutateStatus() }

  return {
    auth: auth ?? null,
    status: status ?? null,
    initialLoading,
    fetchStatus,
  }
}
