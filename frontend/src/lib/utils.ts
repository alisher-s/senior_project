import { format, formatDistanceToNow, isPast } from 'date-fns';

export function formatEventDate(iso: string): string {
  return format(new Date(iso), 'MMM d, yyyy · h:mm a');
}

export function formatRelative(iso: string): string {
  return formatDistanceToNow(new Date(iso), { addSuffix: true });
}

export function isEventPast(iso: string): boolean {
  return isPast(new Date(iso));
}

export function capacityPercent(total: number, available: number): number {
  if (total <= 0) return 100;
  return Math.round(((total - available) / total) * 100);
}

export function capacityColor(total: number, available: number): string {
  const pct = capacityPercent(total, available);
  if (available <= 0) return 'text-nu-error';
  if (pct >= 90) return 'text-nu-error';
  if (pct >= 70) return 'text-nu-warning';
  return 'text-nu-success';
}

export function cn(...classes: (string | false | undefined | null)[]): string {
  return classes.filter(Boolean).join(' ');
}
