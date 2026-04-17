import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { eventsAPI } from '../../api/services';
import { Badge, Button, Spinner, EmptyState } from '../../components/ui/Primitives';
import { formatEventDate } from '../../lib/utils';
import { Plus, CalendarDays, Users, Edit, Eye } from 'lucide-react';
import type { EventDTO } from '../../types';

export default function OrganizerDashboard() {
  // Fetch all events (organizer sees their own via the general list for now)
  const { data, isLoading } = useQuery({
    queryKey: ['events', 'all'],
    queryFn: () => eventsAPI.list({ limit: 100 }).then((r) => r.data),
  });

  const events = data?.items || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="font-display text-2xl font-bold">Organizer Dashboard</h1>
          <p className="text-nu-text-muted text-sm mt-1">Manage your events</p>
        </div>
        <Link to="/organizer/events/new">
          <Button>
            <Plus className="w-4 h-4" /> Create Event
          </Button>
        </Link>
      </div>

      {/* Stats overview */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <StatCard label="Total Events" value={events.length} />
        <StatCard label="Published" value={events.filter((e) => e.status === 'published').length} />
        <StatCard label="Pending Review" value={events.filter((e) => e.moderation_status === 'pending').length} />
        <StatCard
          label="Total Capacity"
          value={events.reduce((sum, e) => sum + e.capacity_total, 0)}
        />
      </div>

      {/* Events list */}
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Spinner className="w-8 h-8" />
        </div>
      ) : events.length === 0 ? (
        <EmptyState
          icon={<CalendarDays className="w-12 h-12" />}
          title="No events yet"
          description="Create your first event to get started."
          action={
            <Link to="/organizer/events/new">
              <Button><Plus className="w-4 h-4" /> Create Event</Button>
            </Link>
          }
        />
      ) : (
        <div className="space-y-3">
          {events.map((event) => (
            <OrganizerEventRow key={event.id} event={event} />
          ))}
        </div>
      )}
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-xl border border-white/5 bg-nu-surface/50 p-4">
      <p className="text-xs text-nu-text-muted">{label}</p>
      <p className="font-display text-2xl font-bold mt-1">{value}</p>
    </div>
  );
}

function OrganizerEventRow({ event }: { event: EventDTO }) {
  const moderationVariant = {
    approved: 'success' as const,
    pending: 'warning' as const,
    rejected: 'error' as const,
  };

  return (
    <div className="flex items-center gap-4 rounded-xl border border-white/5 bg-nu-surface/50 p-4">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <h3 className="font-semibold text-sm truncate">{event.title}</h3>
          <Badge variant={moderationVariant[event.moderation_status]}>
            {event.moderation_status}
          </Badge>
          <Badge variant={event.status === 'published' ? 'success' : event.status === 'cancelled' ? 'error' : 'default'}>
            {event.status}
          </Badge>
        </div>
        <div className="flex items-center gap-4 text-xs text-nu-text-muted">
          <span className="flex items-center gap-1">
            <CalendarDays className="w-3 h-3" />
            {formatEventDate(event.starts_at)}
          </span>
          <span className="flex items-center gap-1">
            <Users className="w-3 h-3" />
            {event.capacity_total - event.capacity_available} / {event.capacity_total}
          </span>
        </div>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <Link to={`/events/${event.id}`}>
          <Button variant="ghost" size="sm"><Eye className="w-3.5 h-3.5" /></Button>
        </Link>
        <Link to={`/organizer/events/${event.id}/edit`}>
          <Button variant="ghost" size="sm"><Edit className="w-3.5 h-3.5" /></Button>
        </Link>
      </div>
    </div>
  );
}
