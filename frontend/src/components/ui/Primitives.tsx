import { type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode } from 'react';
import { cn } from '../../lib/utils';
import { Loader2 } from 'lucide-react';

// ─── Button ───────────────────────────────────────
interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
}

export function Button({
  variant = 'primary',
  size = 'md',
  loading,
  disabled,
  className,
  children,
  ...props
}: ButtonProps) {
  const base = 'inline-flex items-center justify-center font-semibold rounded-xl transition-all duration-200 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-nu-gold/50';
  const variants = {
    primary: 'bg-nu-gold text-nu-dark hover:bg-nu-gold-light',
    secondary: 'bg-nu-surface-light text-nu-text hover:bg-nu-blue border border-white/10',
    danger: 'bg-nu-error/20 text-nu-error hover:bg-nu-error/30 border border-nu-error/30',
    ghost: 'text-nu-text-muted hover:text-nu-text hover:bg-white/5',
  };
  const sizes = {
    sm: 'text-sm px-3 py-1.5 gap-1.5',
    md: 'text-sm px-5 py-2.5 gap-2',
    lg: 'text-base px-7 py-3 gap-2.5',
  };

  return (
    <button
      className={cn(base, variants[variant], sizes[size], className)}
      disabled={disabled || loading}
      {...props}
    >
      {loading && <Loader2 className="w-4 h-4 animate-spin" />}
      {children}
    </button>
  );
}

// ─── Input ────────────────────────────────────────
interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export function Input({ label, error, className, id, ...props }: InputProps) {
  const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label htmlFor={inputId} className="text-sm font-medium text-nu-text-muted">
          {label}
        </label>
      )}
      <input
        id={inputId}
        className={cn(
          'w-full rounded-xl border border-white/10 bg-nu-surface px-4 py-2.5 text-sm text-nu-text placeholder:text-nu-text-muted/50 focus:outline-none focus:ring-2 focus:ring-nu-gold/50 focus:border-nu-gold/50 transition-all',
          error && 'border-nu-error/50 focus:ring-nu-error/50',
          className
        )}
        {...props}
      />
      {error && <p className="text-xs text-nu-error">{error}</p>}
    </div>
  );
}

// ─── Textarea ─────────────────────────────────────
interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
}

export function Textarea({ label, error, className, id, ...props }: TextareaProps) {
  const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label htmlFor={inputId} className="text-sm font-medium text-nu-text-muted">
          {label}
        </label>
      )}
      <textarea
        id={inputId}
        className={cn(
          'w-full rounded-xl border border-white/10 bg-nu-surface px-4 py-2.5 text-sm text-nu-text placeholder:text-nu-text-muted/50 focus:outline-none focus:ring-2 focus:ring-nu-gold/50 focus:border-nu-gold/50 transition-all resize-none',
          error && 'border-nu-error/50 focus:ring-nu-error/50',
          className
        )}
        {...props}
      />
      {error && <p className="text-xs text-nu-error">{error}</p>}
    </div>
  );
}

// ─── Modal ────────────────────────────────────────
interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
}

export function Modal({ open, onClose, title, children }: ModalProps) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md mx-4 bg-nu-navy border border-white/10 rounded-2xl p-6 shadow-2xl animate-[scaleIn_0.2s_ease-out]">
        {title && <h2 className="font-display text-lg font-bold mb-4">{title}</h2>}
        {children}
      </div>
    </div>
  );
}

// ─── Spinner ──────────────────────────────────────
export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={cn('animate-spin text-nu-gold', className)} />;
}

// ─── Badge ────────────────────────────────────────
interface BadgeProps {
  children: ReactNode;
  variant?: 'default' | 'success' | 'warning' | 'error' | 'gold';
}

export function Badge({ children, variant = 'default' }: BadgeProps) {
  const colors = {
    default: 'bg-white/10 text-nu-text-muted',
    success: 'bg-nu-success/15 text-nu-success',
    warning: 'bg-nu-warning/15 text-nu-warning',
    error: 'bg-nu-error/15 text-nu-error',
    gold: 'bg-nu-gold/15 text-nu-gold',
  };
  return (
    <span className={cn('inline-flex items-center px-2.5 py-0.5 rounded-lg text-xs font-semibold', colors[variant])}>
      {children}
    </span>
  );
}

// ─── Empty State ──────────────────────────────────
interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      {icon && <div className="text-nu-text-muted mb-4">{icon}</div>}
      <h3 className="font-display text-lg font-semibold mb-1">{title}</h3>
      {description && <p className="text-sm text-nu-text-muted max-w-sm">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

// ─── Skeleton ─────────────────────────────────────
export function Skeleton({ className }: { className?: string }) {
  return <div className={cn('bg-white/5 rounded-xl animate-pulse', className)} />;
}
