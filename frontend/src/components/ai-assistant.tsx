'use client';

import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useAIQuery } from '@/queries/ai';
import { Logo } from '@/components/brand/logo';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  agent?: string;
  timestamp: Date;
}

export function AIAssistant() {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const scrollRef = useRef<HTMLDivElement>(null);
  const aiQuery = useAIQuery();

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || aiQuery.isPending) return;

    const userMsg: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, userMsg]);
    setInput('');

    try {
      const result = await aiQuery.mutateAsync({ query: userMsg.content });
      const assistantMsg: Message = {
        id: crypto.randomUUID(),
        role: 'assistant',
        content: result.response,
        agent: result.agent,
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, assistantMsg]);
    } catch {
      setMessages((prev) => [
        ...prev,
        {
          id: crypto.randomUUID(),
          role: 'assistant',
          content: 'Sorry, I encountered an error. Please try again.',
          timestamp: new Date(),
        },
      ]);
    }
  };

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="fixed bottom-6 right-6 z-50 flex h-12 w-12 items-center justify-center rounded-full bg-neutral-900 text-white shadow-lg transition-transform hover:scale-105 overflow-hidden"
        aria-label="Open Timeless AI Assistant"
      >
        <Logo size={28} style="mark" variant="white-nobg" className="opacity-95" />
      </button>
    );
  }

  return (
    <div className="fixed bottom-6 right-6 z-50 flex h-[500px] w-[380px] flex-col overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xl dark:border-neutral-800 dark:bg-neutral-950">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
        <div className="flex items-center gap-2">
          <Logo size={22} style="solid" variant="black" />
          <span className="text-sm font-medium text-neutral-900 dark:text-neutral-100">Timeless AI</span>
          <div className="h-2 w-2 rounded-full bg-green-500" />
        </div>
        <button
          onClick={() => setOpen(false)}
          className="text-neutral-400 hover:text-neutral-600"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M18 6 6 18M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Messages */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
        {messages.length === 0 && (
          <div className="flex h-full items-center justify-center">
            <p className="text-sm text-neutral-400">Ask me anything about your sponsors...</p>
          </div>
        )}
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-[85%] rounded-lg px-3 py-2 text-sm ${
                msg.role === 'user'
                  ? 'bg-neutral-900 text-white'
                  : 'bg-neutral-100 text-neutral-800'
              }`}
            >
              {msg.agent && (
                <Badge variant="secondary" className="mb-1 text-[10px]">
                  {msg.agent}
                </Badge>
              )}
              <p className="whitespace-pre-wrap">{msg.content}</p>
            </div>
          </div>
        ))}
        {aiQuery.isPending && (
          <div className="flex justify-start">
            <div className="rounded-lg bg-neutral-100 px-3 py-2">
              <div className="flex gap-1">
                <span className="h-2 w-2 animate-bounce rounded-full bg-neutral-400" />
                <span className="h-2 w-2 animate-bounce rounded-full bg-neutral-400 [animation-delay:0.1s]" />
                <span className="h-2 w-2 animate-bounce rounded-full bg-neutral-400 [animation-delay:0.2s]" />
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Input */}
      <form onSubmit={handleSubmit} className="border-t border-neutral-100 p-3">
        <div className="flex gap-2">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Ask about sponsors, research..."
            className="flex-1 text-sm"
            disabled={aiQuery.isPending}
          />
          <Button type="submit" size="sm" disabled={aiQuery.isPending || !input.trim()}>
            Send
          </Button>
        </div>
      </form>
    </div>
  );
}
