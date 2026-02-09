import { useEffect, useState } from 'react'

interface LoadingScreenProps {
  ready: boolean
}

export function LoadingScreen({ ready }: LoadingScreenProps) {
  const [visible, setVisible] = useState(true)
  const [fadeOut, setFadeOut] = useState(false)

  useEffect(() => {
    if (ready) {
      // Minimum display time so user sees the branding
      const timer = setTimeout(() => {
        setFadeOut(true)
        setTimeout(() => setVisible(false), 500)
      }, 600)
      return () => clearTimeout(timer)
    }
  }, [ready])

  if (!visible) return null

  return (
    <div
      className={`fixed inset-0 z-50 flex items-center justify-center bg-background transition-opacity duration-500 ${
        fadeOut ? 'opacity-0' : 'opacity-100'
      }`}
    >
      <div className="flex flex-col items-center gap-8">
        {/* Logo */}
        <div className="flex items-center gap-1">
          <span className="text-4xl font-bold tracking-tight text-foreground">k8s</span>
          <span className="text-4xl font-bold tracking-tight text-primary">-lab</span>
        </div>

        {/* Animated bar */}
        <div className="w-48 h-1 bg-muted rounded-full overflow-hidden">
          <div className="h-full bg-primary rounded-full animate-loading-bar" />
        </div>

        {/* Status text */}
        <p className="text-sm text-muted-foreground animate-pulse">
          Connecting to cluster...
        </p>
      </div>
    </div>
  )
}
