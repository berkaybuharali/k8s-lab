import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react'

interface LoadingTrackerContextType {
  /** Call when a panel starts loading */
  trackLoading: (id: string) => void
  /** Call when a panel finishes loading */
  trackLoaded: (id: string) => void
  /** True while any tracked panel is still loading */
  isLoading: boolean
  /** Number of panels still loading */
  pendingCount: number
  /** Total panels tracked */
  totalCount: number
}

const LoadingTrackerContext = createContext<LoadingTrackerContextType>({
  trackLoading: () => {},
  trackLoaded: () => {},
  isLoading: false,
  pendingCount: 0,
  totalCount: 0,
})

export function LoadingTrackerProvider({ children }: { children: React.ReactNode }) {
  const [pending, setPending] = useState<Set<string>>(new Set())
  const [total, setTotal] = useState<Set<string>>(new Set())
  // Track which panels have already completed their first load
  const completedRef = useRef<Set<string>>(new Set())

  const trackLoading = useCallback((id: string) => {
    // Only track the first load per panel
    if (completedRef.current.has(id)) return
    setPending(prev => new Set(prev).add(id))
    setTotal(prev => new Set(prev).add(id))
  }, [])

  const trackLoaded = useCallback((id: string) => {
    if (completedRef.current.has(id)) return
    completedRef.current.add(id)
    setPending(prev => {
      const next = new Set(prev)
      next.delete(id)
      return next
    })
  }, [])

  const value = useMemo(() => ({
    trackLoading,
    trackLoaded,
    isLoading: pending.size > 0,
    pendingCount: pending.size,
    totalCount: total.size,
  }), [trackLoading, trackLoaded, pending, total])

  return (
    <LoadingTrackerContext.Provider value={value}>
      {children}
    </LoadingTrackerContext.Provider>
  )
}

export function useLoadingTracker() {
  return useContext(LoadingTrackerContext)
}

/**
 * Hook for panels to register their loading state.
 * Call returned `setLoaded` when data arrives.
 */
export function usePanelLoading(id: string) {
  const { trackLoading, trackLoaded } = useLoadingTracker()
  const registeredRef = useRef(false)

  const setLoading = useCallback(() => {
    if (!registeredRef.current) {
      registeredRef.current = true
      trackLoading(id)
    }
  }, [id, trackLoading])

  const setLoaded = useCallback(() => {
    trackLoaded(id)
  }, [id, trackLoaded])

  return { setLoading, setLoaded }
}
