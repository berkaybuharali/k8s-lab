import { AgentChat } from './AgentChat'
import { Sparkles, ShoppingBag, Heart } from 'lucide-react'

export function ShopPage() {
  return (
    <div className="h-full flex flex-col bg-gradient-to-br from-pink-50/50 to-orange-50/50 dark:from-background dark:to-background">
      {/* Hero Section */}
      <div className="w-full py-12 px-6 text-center space-y-4">
        <div className="inline-flex items-center justify-center p-3 bg-pink-100 dark:bg-pink-900/20 rounded-full mb-4 ring-8 ring-pink-50 dark:ring-pink-900/10">
            <Sparkles className="w-8 h-8 text-pink-500" />
        </div>
        <h1 className="text-4xl md:text-5xl font-bold tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-pink-600 to-orange-500 pb-2">
            Magic Cake Shop
        </h1>
        <p className="text-muted-foreground text-lg max-w-lg mx-auto">
            Design your dream cake with our AI concierge. 
            From flavor to delivery, we handle everything with a sprinkle of magic.
        </p>
      </div>

      {/* Main Content */}
      <div className="flex-1 max-w-5xl mx-auto w-full p-6 grid grid-cols-1 md:grid-cols-[1fr_350px] gap-8 items-start">
        
        {/* Chat Interface */}
        <div className="w-full">
            <div className="bg-card/50 backdrop-blur-sm border rounded-2xl shadow-sm p-1">
                <AgentChat 
                    system="commerce" 
                    className="h-[600px] border-none shadow-none bg-transparent"
                    placeholder="Describe your dream cake..."
                    initialMessage="**Welcome to Magic Cake!** 🍰✨\n\nI can help you design a custom cake. What language would you like to speak?\n\n- English\n- Deutsch\n- Nederlands\n- Türkçe"
                />
            </div>
        </div>

        {/* Sidebar Info */}
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

            <div className="bg-gradient-to-br from-pink-500 to-orange-500 rounded-xl p-6 text-white shadow-lg relative overflow-hidden">
                <div className="absolute top-0 right-0 p-4 opacity-20">
                    <Heart className="w-24 h-24 rotate-12" />
                </div>
                <h3 className="font-bold text-lg mb-2 relative z-10">Amsterdam Special</h3>
                <p className="text-sm opacity-90 relative z-10">
                    Free delivery on all orders over €50 within the ring A10!
                </p>
                <div className="mt-4 text-xs font-mono bg-white/20 inline-block px-2 py-1 rounded">
                    CODE: MAGIC50
                </div>
            </div>
        </div>

      </div>
    </div>
  )
}
