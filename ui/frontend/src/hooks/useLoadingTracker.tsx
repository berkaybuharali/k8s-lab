import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react'

interface PanelEntry {
  id: string
  label: string
  done: boolean
}

interface LoadingTrackerContextType {
  trackLoading: (id: string, label: string) => void
  trackLoaded: (id: string) => void
  isLoading: boolean
  pendingCount: number
  totalCount: number
  /** Labels of panels still loading */
  pendingLabels: string[]
}

const LoadingTrackerContext = createContext<LoadingTrackerContextType>({
  trackLoading: () => {},
  trackLoaded: () => {},
  isLoading: false,
  pendingCount: 0,
  totalCount: 0,
  pendingLabels: [],
})

export function LoadingTrackerProvider({ children }: { children: React.ReactNode }) {
  const [panels, setPanels] = useState<Map<string, PanelEntry>>(new Map())
  const completedRef = useRef<Set<string>>(new Set())

  const trackLoading = useCallback((id: string, label: string) => {
    if (completedRef.current.has(id)) return
    setPanels(prev => {
      const next = new Map(prev)
      next.set(id, { id, label, done: false })
      return next
    })
  }, [])

  const trackLoaded = useCallback((id: string) => {
    if (completedRef.current.has(id)) return
    completedRef.current.add(id)
    setPanels(prev => {
      const next = new Map(prev)
      const entry = next.get(id)
      if (entry) next.set(id, { ...entry, done: true })
      return next
    })
  }, [])

  const value = useMemo(() => {
    const entries = Array.from(panels.values())
    const pending = entries.filter(e => !e.done)
    return {
      trackLoading,
      trackLoaded,
      isLoading: pending.length > 0,
      pendingCount: pending.length,
      totalCount: entries.length,
      pendingLabels: pending.map(e => e.label),
    }
  }, [trackLoading, trackLoaded, panels])

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
 * Hook for panels to register their loading state with a human-readable label.
 */
export function usePanelLoading(id: string, label: string = id) {
  const { trackLoading, trackLoaded } = useLoadingTracker()
  const registeredRef = useRef(false)

  const setLoading = useCallback(() => {
    if (!registeredRef.current) {
      registeredRef.current = true
      trackLoading(id, label)
    }
  }, [id, label, trackLoading])

  const setLoaded = useCallback(() => {
    trackLoaded(id)
  }, [id, trackLoaded])

  return { setLoading, setLoaded }
}
