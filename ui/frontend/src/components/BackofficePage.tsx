/**
 * Phase 5: Magic Cake Backoffice Page
 * Internal operations dashboard for Magic Cake shop
 */

// TODO: Implement BackofficePage with 6 components
//
// Layout: Grid/dashboard style with components:
//
// 1. Revenue Summary (top bar)
//    - Total orders count
//    - Total revenue
//    - Average order value
//    - API: GET /api/orders/stats
//
// 2. Map View Component
//    - Date selector dropdown (next 3 days)
//    - Leaflet.js map with delivery route
//    - Route from Danzigerkade 4 to all delivery addresses
//    - API: GET /api/fulfillment/route?date=YYYY-MM-DD
//    - Markers with order details on hover
//
// 3. Order Table Component
//    - Columns: Order ID, Customer, Cakes (summary), Address, Date, Price, Images
//    - Delete button per row (with confirmation)
//    - Filter by delivery date dropdown
//    - Click to expand: full cake details, full-size images
//    - API: GET /api/orders?date=YYYY-MM-DD, DELETE /api/orders/:id
//
// 4. Inventory Dashboard Component
//    - 5 progress bars for ingredients (chocolate, ananas, banana, walnut, almond)
//    - Color-coded: green (3-5), yellow (2), red (0-1)
//    - Shows: quantity / 5 (max)
//    - Auto-refresh every 30 seconds
//    - API: GET /api/inventory
//
// 5. Agent Activity Log Component
//    - Recent agent interactions feed
//    - Shows: timestamp, system, user query, agent action
//    - Scrollable, max 50 entries
//    - API: GET /api/agent/activity
//
// 6. Agent Chat Panel (collapsed by default)
//    - Embedded <AgentChat system="supply-chain" />
//    - For ad-hoc queries: "How many orders tomorrow?", "List low stock"
//    - Toggle button to expand/collapse
//
// Dependencies:
// - react-leaflet for maps (yarn add react-leaflet leaflet)
// - Date picker for filters
// - Modal/dialog for confirmations
//
// Example structure:
//
// import React, { useState, useEffect } from 'react';
// import { MapContainer, TileLayer, Marker, Polyline } from 'react-leaflet';
// import AgentChat from './AgentChat';
//
// export default function BackofficePage() {
//   const [orders, setOrders] = useState([]);
//   const [inventory, setInventory] = useState({});
//   const [stats, setStats] = useState({});
//   const [selectedDate, setSelectedDate] = useState('2026-02-17');
//
//   useEffect(() => {
//     // Fetch orders, inventory, stats
//   }, [selectedDate]);
//
//   return (
//     <div className="backoffice">
//       <RevenueSummary stats={stats} />
//       <div className="dashboard-grid">
//         <MapView date={selectedDate} />
//         <OrderTable orders={orders} onDelete={handleDelete} />
//         <InventoryDashboard inventory={inventory} />
//         <AgentActivityLog />
//       </div>
//       <AgentChatPanel />
//     </div>
//   );
// }

export default function BackofficePage() {
  return (
    <div>
      <h1>Magic Cake Backoffice</h1>
      <p>Phase 5: Not yet implemented</p>
      <p>This will show:</p>
      <ul>
        <li>Delivery map with route</li>
        <li>Order management table</li>
        <li>Inventory levels</li>
        <li>Revenue stats</li>
        <li>Agent activity log</li>
        <li>Supply Chain agent chat</li>
      </ul>
    </div>
  );
}
