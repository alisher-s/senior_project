import { Link } from 'react-router-dom';
import type { EventDTO } from '../../types';
import { formatEventDate, capacityPercent, capacityColor } from '../../lib/utils';
import { Badge, Skeleton } from '../ui/Primitives';
import { CalendarDays, Users } from 'lucide-react';

interface EventCardProps {
  event: EventDTO;
}

export function EventCard({ event }: EventCardProps) {
  const pct = capacityPercent(event.capacity_total, event.capacity_available);
  const colorClass = capacityColor(event.capacity_total, event.capacity_available);
  const isFull = event.capacity_available <= 0;

  return (
    <Link
      to={`/events/${event.id}`}
      className="group block rounded-2xl border border-white/5 bg-nu-surface/50 hover:bg-nu-surface hover:border-nu-gold/20 transition-all duration-300 overflow-hidden"
    >
      {/* Color accent bar or cover image */}
      {event.cover_image_url ? (
        <div className="h-36 overflow-hidden">
          <img
            src={event.cover_image_url}
            alt={event.title}
            className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
          />
        </div>
      ) : (
        <div className="h-1 bg-gradient-to-r from-nu-gold to-nu-gold-light" />
      )}

      <div className="p-5">
        <div className="flex items-start justify-between gap-3 mb-3">
          <h3 className="font-display font-bold text-base group-hover:text-nu-gold transition-colors line-clamp-2">
            {event.title}
          </h3>
          {isFull && <Badge variant="error">Full</Badge>}
        </div>

        {event.description && (
          <p className="text-sm text-nu-text-muted line-clamp-2 mb-4">
            {event.description}
          </p>
        )}

        <div className="flex flex-col gap-2 text-sm text-nu-text-muted">
          <div className="flex items-center gap-2">
            <CalendarDays className="w-3.5 h-3.5" />
            <span>{formatEventDate(event.starts_at)}</span>
          </div>

          <div className="flex items-center gap-2">
            <Users className="w-3.5 h-3.5" />
            <span className={colorClass}>
              {event.capacity_available} / {event.capacity_total} spots
            </span>
          </div>
        </div>

        {/* Capacity bar */}
        <div className="mt-4 h-1.5 rounded-full bg-white/5 overflow-hidden">
          <div
            className="h-full rounded-full transition-all duration-500 bg-gradient-to-r from-nu-gold to-nu-gold-light"
            style={{ width: `${Math.min(pct, 100)}%` }}
          />
        </div>
      </div>
    </Link>
  );
}

export function EventCardSkeleton() {
  return (
    <div className="rounded-2xl border border-white/5 bg-nu-surface/50 overflow-hidden">
      <Skeleton className="h-1 rounded-none" />
      <div className="p-5 space-y-3">
        <Skeleton className="h-5 w-3/4" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-2/3" />
        <div className="flex gap-4 mt-4">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-4 w-24" />
        </div>
        <Skeleton className="h-1.5 mt-4" />
      </div>
    </div>
  );
}
