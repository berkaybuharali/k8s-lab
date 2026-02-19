import { useState, useRef, useEffect } from 'react'
import { Send, Loader2, User, Bot, CreditCard, X, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import ReactMarkdown from 'react-markdown'

interface Message {
  role: 'user' | 'agent'
  text: string
  timestamp: Date
}

interface AgentChatProps {
  system: 'commerce' | 'supply-chain'
  className?: string
  placeholder?: string
  initialMessage?: string
}

export function AgentChat({ system, className, placeholder = "Type a message...", initialMessage }: AgentChatProps) {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [sessionId, setSessionId] = useState<string>('')
  
  // Payment Modal State
  const [showPaymentModal, setShowPaymentModal] = useState(false)
  const [paymentAmount, setPaymentAmount] = useState<string>('0.00')
  
  const scrollRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, loading])

  // Initial greeting if provided
  useEffect(() => {
    if (initialMessage && messages.length === 0) {
        setMessages([{
            role: 'agent',
            text: initialMessage,
            timestamp: new Date()
        }])
    }
  }, [initialMessage])

  const sendMessage = async (text: string) => {
      setLoading(true)
      try {
        const res = await fetch('/api/agent/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
            system,
            message: text,
            session_id: sessionId
            })
        })

        if (!res.ok) throw new Error('Failed to send message')

        const data = await res.json()
        
        if (data.session_id) {
            setSessionId(data.session_id)
        }

        const responseText = data.response || "I'm having trouble connecting right now."
        const agentMsg: Message = {
            role: 'agent',
            text: responseText,
            timestamp: new Date()
        }
        setMessages(prev => [...prev, agentMsg])

        // Check for payment prompt
        // Look for "proceed with payment" OR ("€" AND "payment")
        const lowerText = responseText.toLowerCase()
        if (lowerText.includes('proceed with payment') || (responseText.includes('€') && lowerText.includes('payment'))) {
            // Extract amount
            const match = responseText.match(/€\s?(\d+\.\d{2})/)
            if (match) {
                setPaymentAmount(match[1])
                setShowPaymentModal(true)
            }
        }

      } catch (err) {
        console.error(err)
        setMessages(prev => [...prev, {
            role: 'agent',
            text: "Error: Could not reach the agent. Is the system running?",
            timestamp: new Date()
        }])
      } finally {
        setLoading(false)
      }
  }

  const handleSend = async () => {
    if (!input.trim() || loading) return

    const userMsg: Message = {
      role: 'user',
      text: input,
      timestamp: new Date()
    }

    setMessages(prev => [...prev, userMsg])
    setInput('')
    
    await sendMessage(userMsg.text)
  }

  const handlePaymentConfirm = async () => {
      setShowPaymentModal(false)
      // Simulate user saying "yes" to the agent's prompt
      const userMsg: Message = {
        role: 'user',
        text: "Yes, please proceed with payment.",
        timestamp: new Date()
      }
      setMessages(prev => [...prev, userMsg])
      await sendMessage("Yes, please proceed with payment.")
  }

  const handlePaymentCancel = async () => {
      setShowPaymentModal(false)
      // Simulate user saying "cancel"
      const userMsg: Message = {
        role: 'user',
        text: "Cancel payment.",
        timestamp: new Date()
      }
      setMessages(prev => [...prev, userMsg])
      await sendMessage("Cancel payment.")
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className={cn("flex flex-col h-[600px] border rounded-xl bg-card shadow-sm overflow-hidden relative", className)}>
      
      {/* Payment Modal Overlay */}
      {showPaymentModal && (
          <div className="absolute inset-0 z-50 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4">
              <div className="bg-card border rounded-xl shadow-2xl max-w-sm w-full p-6 space-y-4 animate-in fade-in zoom-in duration-200">
                  <div className="flex items-center justify-between">
                      <h3 className="text-lg font-bold flex items-center gap-2">
                          <CreditCard className="w-5 h-5 text-primary" />
                          Secure Payment
                      </h3>
                      <button onClick={handlePaymentCancel} className="text-muted-foreground hover:text-foreground">
                          <X className="w-4 h-4" />
                      </button>
                  </div>
                  
                  <div className="space-y-3">
                      <div className="space-y-1">
                          <label className="text-xs font-medium text-muted-foreground">Card Number</label>
                          <div className="flex gap-2">
                              <input type="text" value="4242 4242 4242 4242" readOnly className="flex-1 bg-muted/50 border rounded px-3 py-2 text-sm font-mono" />
                              <div className="w-10 flex items-center justify-center bg-muted/50 border rounded">
                                  <div className="w-6 h-4 bg-orange-500/20 rounded-sm" />
                              </div>
                          </div>
                      </div>
                      <div className="grid grid-cols-2 gap-4">
                          <div className="space-y-1">
                              <label className="text-xs font-medium text-muted-foreground">Expiry</label>
                              <input type="text" value="12/28" readOnly className="w-full bg-muted/50 border rounded px-3 py-2 text-sm font-mono" />
                          </div>
                          <div className="space-y-1">
                              <label className="text-xs font-medium text-muted-foreground">CVV</label>
                              <input type="text" value="123" readOnly className="w-full bg-muted/50 border rounded px-3 py-2 text-sm font-mono" />
                          </div>
                      </div>
                  </div>

                  <div className="pt-2 flex gap-3">
                      <button onClick={handlePaymentCancel} className="flex-1 px-4 py-2 text-sm font-medium border rounded-lg hover:bg-muted transition-colors">
                          Cancel
                      </button>
                      <button onClick={handlePaymentConfirm} className="flex-1 px-4 py-2 text-sm font-medium bg-primary text-primary-foreground rounded-lg hover:opacity-90 transition-opacity flex items-center justify-center gap-2">
                          Pay €{paymentAmount}
                      </button>
                  </div>
                  
                  <p className="text-[10px] text-center text-muted-foreground flex items-center justify-center gap-1">
                      <Check className="w-3 h-3 text-green-500" />
                      Encrypted and secure (Demo Mode)
                  </p>
              </div>
          </div>
      )}

      {/* Header */}
      <div className="px-4 py-3 border-b bg-muted/30 flex items-center justify-between">
        <div className="flex items-center gap-2">
            <span className="font-semibold text-sm">
              {system === 'commerce' ? 'Magic Cake Chat' : 'Supply Chain Agent'}
            </span>
            {sessionId && <div className="w-2 h-2 rounded-full bg-green-500" title="Session active" />}
        </div>
        {sessionId && <span className="text-[10px] text-muted-foreground font-mono">Session: {sessionId.slice(0, 8)}...</span>}
      </div>

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4 bg-muted/10" ref={scrollRef}>
        {messages.length === 0 && !initialMessage && (
            <div className="h-full flex flex-col items-center justify-center text-muted-foreground opacity-50">
                <Bot className="w-12 h-12 mb-2" />
                <p>Start a conversation...</p>
            </div>
        )}
        
        {messages.map((msg, i) => (
          <div key={i} className={cn("flex gap-3", msg.role === 'user' ? "justify-end" : "justify-start")}>
            {msg.role === 'agent' && (
              <div className="w-8 h-8 rounded-full bg-white border flex items-center justify-center shrink-0 shadow-sm">
                <img src="/assets/cake_small_logo_16x16.png" alt="" className="w-4 h-4" />
              </div>
            )}
            
            <div className={cn(
              "max-w-[80%] rounded-2xl px-4 py-2 text-sm prose prose-sm dark:prose-invert max-w-none break-words",
              msg.role === 'user' 
                ? "bg-primary text-primary-foreground rounded-tr-none" 
                : "bg-muted border rounded-tl-none"
            )}>
              <ReactMarkdown 
                components={{
                    // Custom renderer for images to support cake previews
                    img: ({...props}) => (
                        <img {...props} className="rounded-lg max-w-full h-auto mt-2 border" alt={props.alt || "Agent image"} />
                    ),
                    p: ({...props}) => <p {...props} className="m-0" />
                }}
              >
                {msg.text}
              </ReactMarkdown>
              <div className="text-[10px] opacity-50 mt-1 text-right">
                {msg.timestamp.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}
              </div>
            </div>

            {msg.role === 'user' && (
              <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center shrink-0">
                <User className="w-4 h-4 text-primary-foreground" />
              </div>
            )}
          </div>
        ))}

        {loading && (
          <div className="flex gap-3 justify-start">
            <div className="w-8 h-8 rounded-full bg-white border flex items-center justify-center shrink-0 shadow-sm">
                <img src="/assets/cake_small_logo_16x16.png" alt="" className="w-4 h-4" />
            </div>
            <div className="bg-muted border rounded-2xl rounded-tl-none px-4 py-3 flex items-center gap-1">
              <div className="w-1.5 h-1.5 bg-foreground/40 rounded-full animate-bounce [animation-delay:-0.3s]" />
              <div className="w-1.5 h-1.5 bg-foreground/40 rounded-full animate-bounce [animation-delay:-0.15s]" />
              <div className="w-1.5 h-1.5 bg-foreground/40 rounded-full animate-bounce" />
            </div>
          </div>
        )}
      </div>

      {/* Input Area */}
      <div className="p-4 bg-card border-t">
        <div className="relative flex items-center">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            className="w-full min-h-[50px] max-h-[150px] pl-4 pr-12 py-3 rounded-xl border bg-muted/30 focus:bg-background focus:ring-1 focus:ring-primary resize-none text-sm"
            rows={1}
          />
          <button 
            onClick={handleSend}
            disabled={!input.trim() || loading}
            className={cn(
              "absolute right-2 p-2 rounded-lg transition-all",
              input.trim() && !loading 
                ? "bg-primary text-primary-foreground hover:opacity-90 shadow-sm" 
                : "text-muted-foreground hover:bg-muted"
            )}
          >
            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
          </button>
        </div>
        <div className="text-[10px] text-center text-muted-foreground mt-2">
            AI can make mistakes. Please check important info.
        </div>
      </div>
    </div>
  )
}
