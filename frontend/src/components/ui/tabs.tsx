import * as React from 'react';
import { cn } from '@/lib/utils';

interface TabsProps {
  value: string;
  onValueChange: (value: string) => void;
  children: React.ReactNode;
  className?: string;
}

export function Tabs({ value, onValueChange, children, className }: TabsProps) {
  return (
    <div className={cn('w-full', className)} data-value={value}>
      {React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
          return React.cloneElement(child as React.ReactElement<any>, { value, onValueChange });
        }
        return child;
      })}
    </div>
  );
}

interface TabsListProps {
  children: React.ReactNode;
  className?: string;
}

export function TabsList({ children, className }: TabsListProps) {
  return (
    <div
      className={cn(
        'inline-flex h-9 items-center justify-start gap-1 rounded-lg bg-neutral-100 p-1',
        className
      )}
    >
      {children}
    </div>
  );
}

interface TabsTriggerProps {
  value?: string;
  onValueChange?: (value: string) => void;
  triggerValue: string;
  children: React.ReactNode;
  className?: string;
}

export function TabsTrigger({ value, onValueChange, triggerValue, children, className }: TabsTriggerProps) {
  const isActive = value === triggerValue;
  return (
    <button
      onClick={() => onValueChange?.(triggerValue)}
      className={cn(
        'inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium transition-all',
        isActive
          ? 'bg-white text-neutral-900 shadow-sm'
          : 'text-neutral-500 hover:text-neutral-700',
        className
      )}
    >
      {children}
    </button>
  );
}

interface TabsContentProps {
  value?: string;
  contentValue: string;
  children: React.ReactNode;
  className?: string;
}

export function TabsContent({ value, contentValue, children, className }: TabsContentProps) {
  if (value !== contentValue) return null;
  return <div className={cn('mt-3', className)}>{children}</div>;
}
