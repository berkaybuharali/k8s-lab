import { useEffect, useRef, useState } from 'react'
import { ChevronDown } from 'lucide-react'

declare global {
  interface Window { google: any }
}

const FULFILLMENT = { lat: 52.3962, lng: 4.8763 }

interface Order {
  order_id: string
  customer_name: string
  address: string
  postcode: string
  delivery_date: string
  status: string
}

export function BackofficeMap() {
  const mapContainer = useRef<HTMLDivElement>(null)
  const mapRef = useRef<any>(null)
  const markersRef = useRef<any[]>([])
  const polylineRef = useRef<any>(null)

  const [mapsReady, setMapsReady] = useState(false)
  const [noKey, setNoKey] = useState(false)
  const [orders, setOrders] = useState<Order[]>([])
  const [dates, setDates] = useState<string[]>([])
  const [selectedDate, setSelectedDate] = useState<string>('')
  const [geocoding, setGeocoding] = useState(false)

  // Load Google Maps SDK once
  useEffect(() => {
    let mounted = true
    fetch('/api/maps-key')
      .then(r => r.json())
      .then(({ key }) => {
        if (!key) { setNoKey(true); return }
        if (window.google?.maps) { if (mounted) setMapsReady(true); return }
        const script = document.createElement('script')
        script.src = `https://maps.googleapis.com/maps/api/js?key=${key}`
        script.async = true
        script.onload = () => { if (mounted) setMapsReady(true) }
        script.onerror = () => { if (mounted) setNoKey(true) }
        document.head.appendChild(script)
      })
      .catch(() => { if (mounted) setNoKey(true) })
    return () => { mounted = false }
  }, [])

  // Fetch orders and extract available delivery dates
  useEffect(() => {
    fetch('/api/orders')
      .then(r => r.json())
      .then((data: Order[]) => {
        if (!Array.isArray(data)) return
        setOrders(data)
        const uniqueDates = [...new Set(data.map(o => o.delivery_date).filter(Boolean))].sort()
        setDates(uniqueDates)
        if (uniqueDates.length > 0) setSelectedDate(uniqueDates[0])
      })
      .catch(() => {})
  }, [])

  // Init map once SDK is ready
  useEffect(() => {
    if (!mapsReady || !mapContainer.current || mapRef.current) return
    mapRef.current = new window.google.maps.Map(mapContainer.current, {
      center: FULFILLMENT,
      zoom: 13,
      mapTypeControl: false,
      streetViewControl: false,
      fullscreenControl: false,
    })

    // Fulfillment center marker — always visible
    new window.google.maps.Marker({
      position: FULFILLMENT,
      map: mapRef.current,
      title: 'Magic Cake Fulfillment — Danzigerkade 4',
      icon: {
        path: window.google.maps.SymbolPath.CIRCLE,
        scale: 11,
        fillColor: '#3b82f6',
        fillOpacity: 1,
        strokeColor: '#ffffff',
        strokeWeight: 2,
      },
    })
  }, [mapsReady])

  // Update delivery markers whenever selected date or map changes
  useEffect(() => {
    if (!mapsReady || !mapRef.current) return

    // Clear previous markers and polyline
    markersRef.current.forEach(m => m.setMap(null))
    markersRef.current = []
    if (polylineRef.current) { polylineRef.current.setMap(null); polylineRef.current = null }

    const dayOrders = orders.filter(o => o.delivery_date === selectedDate && o.address)
    if (dayOrders.length === 0) {
      mapRef.current.setCenter(FULFILLMENT)
      mapRef.current.setZoom(13)
      return
    }

    setGeocoding(true)
    const geocoder = new window.google.maps.Geocoder()
    const coords: Array<{ lat: number; lng: number }> = []
    let resolved = 0

    dayOrders.forEach((order, i) => {
      const query = `${order.address}, ${order.postcode}, Amsterdam, Netherlands`
      geocoder.geocode({ address: query }, (results: any[], status: string) => {
        resolved++
        if (status === 'OK' && results[0]) {
          const pos = results[0].geometry.location
          coords.push({ lat: pos.lat(), lng: pos.lng() })

          const marker = new window.google.maps.Marker({
            position: pos,
            map: mapRef.current,
            title: `#${i + 1} — ${order.customer_name}`,
            label: { text: String(i + 1), color: '#ffffff', fontSize: '11px', fontWeight: 'bold' },
            icon: {
              path: window.google.maps.SymbolPath.CIRCLE,
              scale: 14,
              fillColor: '#ef4444',
              fillOpacity: 1,
              strokeColor: '#ffffff',
              strokeWeight: 2,
            },
          })
          new window.google.maps.InfoWindow({
            content: `<div style="font-size:12px;line-height:1.5"><strong>${order.customer_name}</strong><br>${order.address}<br>${order.postcode}</div>`
          }).open({ anchor: marker, map: mapRef.current, shouldFocus: false })
          marker.addListener('click', () => {})
          markersRef.current.push(marker)
        }

        if (resolved === dayOrders.length) {
          setGeocoding(false)
          if (coords.length > 0) {
            const path = [FULFILLMENT, ...coords]
            polylineRef.current = new window.google.maps.Polyline({
              path,
              map: mapRef.current,
              strokeColor: '#3b82f6',
              strokeWeight: 3,
              strokeOpacity: 0.8,
              icons: [{ icon: { path: 'M 0,-1 0,1', strokeOpacity: 1, scale: 4 }, offset: '0', repeat: '20px' }],
            })
            const bounds = new window.google.maps.LatLngBounds()
            path.forEach(p => bounds.extend(p))
            mapRef.current.fitBounds(bounds, 60)
          }
        }
      })
    })
  }, [mapsReady, selectedDate, orders])

  const dayOrders = orders.filter(o => o.delivery_date === selectedDate)

  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden h-[400px] relative">
      {/* Date selector overlay */}
      <div className="absolute top-3 right-3 z-[400]">
        {dates.length > 0 ? (
          <div className="relative">
            <select
              value={selectedDate}
              onChange={e => setSelectedDate(e.target.value)}
              className="appearance-none bg-background/90 backdrop-blur border rounded-md text-xs font-medium px-3 py-1.5 pr-7 shadow-sm focus:outline-none cursor-pointer"
            >
              {dates.map(d => (
                <option key={d} value={d}>{d} ({orders.filter(o => o.delivery_date === d).length} deliveries)</option>
              ))}
            </select>
            <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 w-3 h-3 pointer-events-none text-muted-foreground" />
          </div>
        ) : (
          <div className="bg-background/90 backdrop-blur px-3 py-1 rounded-md text-xs font-medium border shadow-sm text-muted-foreground">
            Delivery Route
          </div>
        )}
      </div>

      <div ref={mapContainer} className="w-full h-full" />

      {/* No key */}
      {noKey && (
        <div className="absolute inset-0 flex items-center justify-center bg-muted/30 z-[500]">
          <span className="text-sm text-muted-foreground text-center px-6">
            Set <code className="bg-muted px-1 rounded font-mono">GOOGLE_API_KEY</code> env variable before running the UI to enable Google Maps.
          </span>
        </div>
      )}

      {/* Geocoding spinner */}
      {geocoding && (
        <div className="absolute bottom-3 left-3 z-[400] bg-background/90 backdrop-blur px-3 py-1 rounded-md text-xs border shadow-sm text-muted-foreground animate-pulse">
          Placing stops...
        </div>
      )}

      {/* No orders for selected date */}
      {mapsReady && !noKey && dates.length > 0 && dayOrders.length === 0 && (
        <div className="absolute bottom-3 left-3 z-[400] bg-background/90 backdrop-blur px-3 py-1 rounded-md text-xs border shadow-sm text-muted-foreground">
          No deliveries for this date
        </div>
      )}

      {/* No order data at all */}
      {mapsReady && !noKey && dates.length === 0 && (
        <div className="absolute bottom-3 left-3 z-[400] bg-background/90 backdrop-blur px-3 py-1 rounded-md text-xs border shadow-sm text-muted-foreground">
          No order data — run Load Agent Data to seed orders
        </div>
      )}
    </div>
  )
}
