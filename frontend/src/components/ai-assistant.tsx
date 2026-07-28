'use client';

import { useState, useRef, useEffect } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useAIQuery } from '@/queries/ai';
import { useDashboardStats } from '@/queries/analytics';
import { useIntegrations } from '@/queries/integrations';
import { useReducedMotion } from '@/hooks/use-media-query';
import { Logo } from '@/components/brand/logo';
import RotatingText from '@/components/ui/rotating-text';
import ReactMarkdown from 'react-markdown';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  agent?: string;
  timestamp: Date;
}

const QUICK_ACTIONS = [
  { label: 'Research sponsors', query: 'Find new sponsor companies that could be a good fit.' },
  { label: 'Clean my CRM', query: 'Review my CRM for duplicates or stale records and clean it up.' },
  { label: 'Draft follow-ups', query: 'Draft follow-up emails for my most recent outreach.' },
  { label: 'Summarize yesterday', query: "Summarize what happened in my workspace yesterday." },
];

function greetingForTimeOfDay() {
  const hour = new Date().getHours();
  if (hour < 12) return 'Good morning';
  if (hour < 18) return 'Good afternoon';
  return 'Good evening';
}

// Desktop/tablet: a small persistent floating panel (unchanged from before).
// Phone: the same panel becomes a full-screen workspace so typing, reading
// replies, and tapping quick actions all get room to breathe.
export function AIAssistant() {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [greetingId, setGreetingId] = useState<string | null>(null);
  const [greetingVariants, setGreetingVariants] = useState<string[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);
  const aiQuery = useAIQuery();
  const { data: statsData } = useDashboardStats();
  const { data: integrationsData } = useIntegrations();
  const prefersReducedMotion = useReducedMotion();

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  useEffect(() => {
    if (!open || messages.length > 0 || !statsData) return;

    const stats = statsData.data;
    const activeIntegrations = (integrationsData?.data ?? []).filter((i) => i.status === 'active').length;

    const parts = [
      `${stats.total_companies} ${stats.total_companies === 1 ? 'company' : 'companies'}`,
      `${stats.total_contacts} ${stats.total_contacts === 1 ? 'contact' : 'contacts'}`,
      `${stats.total_sponsors} active ${stats.total_sponsors === 1 ? 'sponsor' : 'sponsors'}`,
    ];
    if (activeIntegrations > 0) {
      parts.push(`${activeIntegrations} connected ${activeIntegrations === 1 ? 'integration' : 'integrations'}`);
    }

    const greeting = greetingForTimeOfDay();
    const isEmpty = stats.total_companies + stats.total_contacts + stats.total_sponsors === 0;

    const variants = isEmpty
      ? [
          `${greeting}. I haven't found much in your workspace yet — connect an integration or ask me to start researching.`,
        ]
      : [
          `${greeting}. I found ${parts.join(', ')}. Here's what I can help with today.`,
          `${greeting}. Your workspace has ${stats.total_companies} ${
            stats.total_companies === 1 ? 'company' : 'companies'
          } and ${stats.total_sponsors} active ${
            stats.total_sponsors === 1 ? 'sponsor' : 'sponsors'
          } in play — want me to suggest today's priorities?`,
        ];

    const id = crypto.randomUUID();
    setGreetingId(id);
    setGreetingVariants(variants);
    setMessages([
      {
        id,
        role: 'assistant',
        content: variants[0],
        timestamp: new Date(),
      },
    ]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, statsData, integrationsData]);

  const submitQuery = async (query: string) => {
    if (!query.trim() || aiQuery.isPending) return;

    const userMsg: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content: query.trim(),
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await submitQuery(input);
  };

  return (
    <>
      {!open && (
        <button
          onClick={() => setOpen(true)}
          aria-label="Open Timeless AI Assistant"
          className="fixed bottom-24 right-4 z-40 flex h-14 w-14 items-center justify-center overflow-hidden rounded-full bg-neutral-900 text-white shadow-lg transition-transform hover:scale-105 md:bottom-6 md:right-6 md:h-12 md:w-12"
        >
          <Logo size={28} style="mark" variant="white-nobg" className="opacity-95" />
        </button>
      )}

      <AnimatePresence>
        {open && (
          <motion.div
            className="fixed inset-0 z-50 flex flex-col bg-white dark:bg-neutral-950 md:inset-auto md:bottom-6 md:right-6 md:h-[500px] md:w-[380px] md:overflow-hidden md:rounded-xl md:border md:border-neutral-200 md:shadow-2xl md:dark:border-neutral-800"
            initial={{ opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 24 }}
            transition={prefersReducedMotion ? { duration: 0 } : { type: 'spring', damping: 28, stiffness: 320 }}
          >
            {/* Header */}
            <div
              className="flex items-center justify-between border-b border-neutral-100 px-4 pb-3 dark:border-neutral-800 md:pt-3"
              style={{ paddingTop: 'max(0.75rem, env(safe-area-inset-top, 0px))' }}
            >
              <div className="flex items-center gap-2">
                <Logo size={22} style="solid" variant="black" />
                <span className="text-sm font-medium text-neutral-900 dark:text-neutral-100">Timeless AI</span>
                <div className="h-2 w-2 rounded-full bg-green-500" />
              </div>
              <button
                onClick={() => setOpen(false)}
                aria-label="Close AI Assistant"
                className="flex h-9 w-9 items-center justify-center text-neutral-400 hover:text-neutral-600"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M18 6 6 18M6 6l12 12" />
                </svg>
              </button>
            </div>

            {/* Messages */}
            <div ref={scrollRef} className="flex-1 space-y-3 overflow-y-auto overscroll-contain px-4 py-3">
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
                    className={`max-w-[88%] rounded-lg px-3 py-2 text-sm sm:max-w-[85%] ${
                      msg.role === 'user'
                        ? 'bg-neutral-900 text-white dark:bg-neutral-100 dark:text-neutral-900'
                        : 'bg-neutral-100 text-neutral-800 dark:bg-neutral-800 dark:text-neutral-100'
                    }`}
                  >
                    {msg.agent && (
                      <Badge variant="secondary" className="mb-1 text-[10px]">
                        {msg.agent}
                      </Badge>
                    )}
                    {msg.id === greetingId && messages.length === 1 && greetingVariants.length > 1 ? (
                      <RotatingText
                        texts={greetingVariants}
                        auto={!prefersReducedMotion}
                        rotationInterval={6000}
                        staggerDuration={0.015}
                        splitBy="words"
                        mainClassName="text-sm text-neutral-800 dark:text-neutral-100"
                        transition={{ type: 'spring', damping: 25, stiffness: 300 }}
                      />
                    ) : (
                      <div className="prose prose-sm prose-neutral dark:prose-invert max-w-none [&>*:first-child]:mt-0 [&>*:last-child]:mb-0">
                        <ReactMarkdown>{msg.content}</ReactMarkdown>
                      </div>
                    )}
                  </div>
                </div>
              ))}
              {aiQuery.isPending && (
                <div className="flex justify-start">
                  <div className="rounded-lg bg-neutral-100 px-3 py-2 dark:bg-neutral-800">
                    <div className="flex gap-1">
                      <span className="h-2 w-2 animate-bounce rounded-full bg-neutral-400 dark:bg-neutral-500" />
                      <span className="h-2 w-2 animate-bounce rounded-full bg-neutral-400 dark:bg-neutral-500 [animation-delay:0.1s]" />
                      <span className="h-2 w-2 animate-bounce rounded-full bg-neutral-400 dark:bg-neutral-500 [animation-delay:0.2s]" />
                    </div>
                  </div>
                </div>
              )}
              {messages.length <= 1 && !aiQuery.isPending && (
                <div className="flex flex-wrap gap-1.5 pt-1">
                  {QUICK_ACTIONS.map((action) => (
                    <button
                      key={action.label}
                      onClick={() => submitQuery(action.query)}
                      className="min-h-[36px] rounded-full border border-neutral-200 px-2.5 py-1 text-xs text-neutral-600 transition-colors hover:bg-neutral-100 dark:border-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-900"
                    >
                      {action.label}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Input */}
            <form
              onSubmit={handleSubmit}
              className="border-t border-neutral-100 p-3 dark:border-neutral-800"
              style={{ paddingBottom: 'max(0.75rem, env(safe-area-inset-bottom, 0px))' }}
            >
              <div className="flex gap-2">
                <Input
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  placeholder="Ask about sponsors, research..."
                  className="h-11 flex-1 text-sm md:h-9"
                  disabled={aiQuery.isPending}
                />
                <Button type="submit" size="sm" className="h-11 md:h-9" disabled={aiQuery.isPending || !input.trim()}>
                  Send
                </Button>
              </div>
            </form>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
}
