/**
 * Phase 5: Reusable Agent Chat Component
 * Used by both ShopPage (commerce) and BackofficePage (supply-chain)
 */

// TODO: Implement AgentChat component
//
// Props:
//   system: "commerce" | "supply-chain"
//
// Features:
// - Message input field
// - Chat history display (user messages + agent responses)
// - Markdown rendering for agent responses
// - Image display support (for cake previews from Imagen)
// - Session management (generates session ID, maintains conversation)
// - Loading states (typing indicator while agent processes)
// - Error handling (display errors inline)
//
// API:
//   POST /api/agent/chat
//   Body: {system, message, session_id}
//   Response: {response, session_id}
//
// State:
// - messages: [{role: "user"|"agent", content: string, images?: string[]}]
// - sessionId: string (generated on mount, persisted in component)
// - loading: boolean
// - error: string | null
//
// Example structure:
//
// import React, { useState, useEffect, useRef } from 'react';
// import ReactMarkdown from 'react-markdown';
//
// interface Message {
//   role: 'user' | 'agent';
//   content: string;
//   images?: string[];
//   timestamp: Date;
// }
//
// interface AgentChatProps {
//   system: 'commerce' | 'supply-chain';
// }
//
// export default function AgentChat({ system }: AgentChatProps) {
//   const [messages, setMessages] = useState<Message[]>([]);
//   const [input, setInput] = useState('');
//   const [loading, setLoading] = useState(false);
//   const [sessionId] = useState(() => generateSessionId());
//   const messagesEndRef = useRef<HTMLDivElement>(null);
//
//   const sendMessage = async () => {
//     if (!input.trim()) return;
//
//     const userMessage = { role: 'user', content: input, timestamp: new Date() };
//     setMessages(prev => [...prev, userMessage]);
//     setInput('');
//     setLoading(true);
//
//     try {
//       const response = await fetch('/api/agent/chat', {
//         method: 'POST',
//         headers: { 'Content-Type': 'application/json' },
//         body: JSON.stringify({ system, message: input, session_id: sessionId })
//       });
//       const data = await response.json();
//
//       const agentMessage = {
//         role: 'agent',
//         content: data.response,
//         images: data.images, // If agent sends image URLs
//         timestamp: new Date()
//       };
//       setMessages(prev => [...prev, agentMessage]);
//     } catch (error) {
//       console.error('Chat error:', error);
//       // Show error message
//     } finally {
//       setLoading(false);
//     }
//   };
//
//   useEffect(() => {
//     messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
//   }, [messages]);
//
//   return (
//     <div className="agent-chat">
//       <div className="messages">
//         {messages.map((msg, i) => (
//           <div key={i} className={`message message-${msg.role}`}>
//             <ReactMarkdown>{msg.content}</ReactMarkdown>
//             {msg.images?.map(url => <img key={url} src={url} alt="Cake" />)}
//           </div>
//         ))}
//         {loading && <div className="typing-indicator">Agent is typing...</div>}
//         <div ref={messagesEndRef} />
//       </div>
//
//       <div className="input-area">
//         <input
//           value={input}
//           onChange={e => setInput(e.target.value)}
//           onKeyPress={e => e.key === 'Enter' && sendMessage()}
//           placeholder="Type your message..."
//           disabled={loading}
//         />
//         <button onClick={sendMessage} disabled={loading || !input.trim()}>
//           Send
//         </button>
//       </div>
//     </div>
//   );
// }
//
// function generateSessionId(): string {
//   return `session-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
// }

interface AgentChatProps {
  system: 'commerce' | 'supply-chain';
}

export default function AgentChat({ system }: AgentChatProps) {
  return (
    <div>
      <h3>Agent Chat: {system}</h3>
      <p>Phase 5: Not yet implemented</p>
      <p>This will be a conversational interface with the {system} agent.</p>
    </div>
  );
}
