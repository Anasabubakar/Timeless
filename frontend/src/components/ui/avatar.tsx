import * as React from 'react';
import { cn } from '@/lib/utils';

interface AvatarProps extends React.HTMLAttributes<HTMLDivElement> {
  src?: string | null;
  alt?: string;
  fallback?: string;
  size?: 'sm' | 'md' | 'lg';
}

const sizeClasses = {
  sm: 'h-7 w-7 text-xs',
  md: 'h-9 w-9 text-sm',
  lg: 'h-11 w-11 text-base',
};

export function Avatar({ src, alt, fallback, size = 'md', className, ...props }: AvatarProps) {
  const [imgError, setImgError] = React.useState(false);

  const initials = fallback || (alt ? alt.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase() : '?');

  if (src && !imgError) {
    return (
      <div
        className={cn('relative overflow-hidden rounded-full', sizeClasses[size], className)}
        {...props}
      >
        <img
          src={src}
          alt={alt || ''}
          onError={() => setImgError(true)}
          className="h-full w-full object-cover"
        />
      </div>
    );
  }

  return (
    <div
      className={cn(
        'flex items-center justify-center rounded-full bg-neutral-100 font-medium text-neutral-600',
        sizeClasses[size],
        className
      )}
      {...props}
    >
      {initials}
    </div>
  );
}
