import { useEffect, useRef, useState } from 'react'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

// Custom marker icon to avoid 404s with default leaflet markers in some bundlers
const icon = L.divIcon({
  className: 'custom-marker',
  html: `<div style="background-color: #ef4444; width: 12px; height: 12px; border-radius: 50%; border: 2px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);"></div>`,
  iconSize: [12, 12],
  iconAnchor: [6, 6]
})

const hubIcon = L.divIcon({
  className: 'hub-marker',
  html: `<div style="background-color: #3b82f6; width: 16px; height: 16px; border-radius: 50%; border: 2px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);"></div>`,
  iconSize: [16, 16],
  iconAnchor: [8, 8]
})

export function BackofficeMap() {
  const mapContainer = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<L.Map | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!mapContainer.current || mapInstance.current) return

    // Init map centered on Amsterdam
    const map = L.map(mapContainer.current).setView([52.3676, 4.9041], 13)
    mapInstance.current = map

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; OpenStreetMap contributors'
    }).addTo(map)

    // Fulfillment Center (Danzigerkade 4)
    L.marker([52.3962, 4.8763], { icon: hubIcon })
      .addTo(map)
      .bindPopup("<b>Magic Cake Fulfillment</b><br>Danzigerkade 4")

    // Fetch route
    fetch('/api/fulfillment/route')
      .then(res => res.json())
      .then(() => {
        // Mock points for now if API returns strings, or if it returns coords
        // Phase 6.1 placeholder API returns list of address strings
        // We'll hardcode some coords for visual demo matching the seeded data
        
        // Random locations around Amsterdam for demo
        const stops = [
          [52.3650, 4.8920], // Herengracht
          [52.3730, 4.8930], // Damrak
          [52.3590, 4.9080], // Weesperstraat
          [52.3800, 4.8850]  // Haarlemmerdijk
        ] as L.LatLngTuple[]

        stops.forEach((stop, i) => {
           L.marker(stop, { icon }).addTo(map).bindPopup(`Delivery #${i+1}`)
        })
        
        // Draw polyline
        const path = [[52.3962, 4.8763] as L.LatLngTuple, ...stops]
        L.polyline(path, { color: '#3b82f6', weight: 3, opacity: 0.7, dashArray: '5, 10' }).addTo(map)
        
        // Fit bounds
        const bounds = L.latLngBounds(path)
        map.fitBounds(bounds, { padding: [50, 50] })
      })
      .catch(console.error)
      .finally(() => setLoading(false))

    return () => {
      map.remove()
      mapInstance.current = null
    }
  }, [])

  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden h-[400px] relative">
       <div className="absolute top-3 right-3 z-[400] bg-background/90 backdrop-blur px-3 py-1 rounded-md text-xs font-medium border shadow-sm">
         Delivery Route (Today)
       </div>
       <div ref={mapContainer} className="w-full h-full bg-muted/20" />
       {loading && (
         <div className="absolute inset-0 flex items-center justify-center bg-background/50 backdrop-blur-sm z-[500]">
           <span className="text-sm text-muted-foreground animate-pulse">Loading Map...</span>
         </div>
       )}
    </div>
  )
}
