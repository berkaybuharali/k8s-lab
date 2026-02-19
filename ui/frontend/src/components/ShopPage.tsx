import { AgentChat } from './AgentChat'
import { ShoppingBag } from 'lucide-react'

export function ShopPage() {
  return (
    <div className="h-full flex flex-col overflow-y-auto">
      {/* Amsterdam banner — page starts here */}
      <div className="max-w-5xl mx-auto w-full px-6 pt-6 pb-2">
        <div className="rounded-xl overflow-hidden shadow-sm">
          <img
            src="/assets/cake_amsterdam_banner_new_728x90.png"
            alt="Magic Cake Amsterdam"
            className="w-full block"
          />
        </div>
        <p className="text-sm text-center text-muted-foreground mt-2" style={{ fontFamily: "'Playfair Display', Georgia, serif", fontStyle: 'italic', fontWeight: 300 }}>
          Designed by AI. Baked with love.
        </p>
      </div>

      {/* Main Content */}
      <div className="flex-1 max-w-5xl mx-auto w-full px-6 pt-4 pb-6 grid grid-cols-1 md:grid-cols-[1fr_350px] gap-8 items-start">

        {/* Chat Interface */}
        <div className="w-full">
          <div className="border rounded-2xl shadow-sm overflow-hidden bg-card">
            <AgentChat
              system="commerce"
              className="h-[600px] border-none shadow-none bg-transparent"
              placeholder="Describe your dream cake..."
              initialMessage={`**Welcome to Magic Cake!** 🍰✨

I can help you design a custom cake. What language would you like to speak?

- English
- Deutsch
- Nederlands
- Türkçe`}
            />
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6 hidden md:block">
          <div className="bg-card border rounded-xl p-6 shadow-sm space-y-4">
            <div className="flex items-center gap-2 font-semibold text-lg">
              <ShoppingBag className="w-5 h-5 text-primary" />
              <span>How it works</span>
            </div>
            <ol className="space-y-4 text-sm text-muted-foreground relative border-l ml-2 pl-6">
              <li className="relative">
                <span className="absolute -left-[29px] w-5 h-5 rounded-full bg-primary/10 flex items-center justify-center text-[10px] font-bold text-primary ring-4 ring-background">1</span>
                <strong className="text-foreground">Choose Language</strong>
                <p>We speak English, Dutch, German, and Turkish.</p>
              </li>
              <li className="relative">
                <span className="absolute -left-[29px] w-5 h-5 rounded-full bg-primary/10 flex items-center justify-center text-[10px] font-bold text-primary ring-4 ring-background">2</span>
                <strong className="text-foreground">Design Cake</strong>
                <p>Pick flavors, nuts, and a theme. Our AI will generate a preview!</p>
              </li>
              <li className="relative">
                <span className="absolute -left-[29px] w-5 h-5 rounded-full bg-primary/10 flex items-center justify-center text-[10px] font-bold text-primary ring-4 ring-background">3</span>
                <strong className="text-foreground">Checkout</strong>
                <p>Confirm delivery address in Amsterdam and pay securely.</p>
              </li>
            </ol>
          </div>

          {/* Amsterdam promotional banner */}
          <div className="rounded-xl overflow-hidden border shadow-sm">
            <img
              src="/assets/cake_square_banner_336x280.png"
              alt="Magic Cake Amsterdam"
              className="w-full block object-cover"
            />
            <div className="bg-card px-4 py-3 text-sm text-muted-foreground border-t">
              Free delivery on all orders over <span className="font-semibold text-foreground">€50</span>.
            </div>
          </div>
        </div>

      </div>

      {/* Leaderboard footer banner — full content width */}
      <div className="max-w-5xl mx-auto w-full px-6 pb-8">
        <div className="rounded-xl overflow-hidden shadow-sm">
          <img
            src="/assets/cake_horizontal_banner_728x90.png"
            alt=""
            className="w-full block"
          />
        </div>
      </div>
    </div>
  )
}
