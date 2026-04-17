import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { eventsAPI, type EventFilters as Filters } from '../../api/services';
import { EventCard, EventCardSkeleton } from '../../components/events/EventCard';
import EventFilters from '../../components/events/EventFilters';
import { EmptyState, Button } from '../../components/ui/Primitives';
import { CalendarDays, ChevronLeft, ChevronRight } from 'lucide-react';

const PAGE_SIZE = 12;

export default function EventsPage() {
  const [filters, setFilters] = useState<Filters>({});
  const [page, setPage] = useState(0);

  const queryParams = {
    ...filters,
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
  };

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['events', queryParams],
    queryFn: () => eventsAPI.list(queryParams).then((r) => r.data),
  });

  const events = data?.items || [];
  const hasMore = events.length === PAGE_SIZE;

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-bold">Events</h1>
          <p className="text-nu-text-muted text-sm mt-1">Browse upcoming events at Nazarbayev University</p>
        </div>
      </div>

      <EventFilters
        onFilterChange={(f) => {
          setFilters(f);
          setPage(0);
        }}
      />

      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <EventCardSkeleton key={i} />
          ))}
        </div>
      ) : isError ? (
        <EmptyState
          title="Failed to load events"
          description="Something went wrong. Please try again."
          action={<Button onClick={() => refetch()} variant="secondary">Retry</Button>}
        />
      ) : events.length === 0 ? (
        <EmptyState
          icon={<CalendarDays className="w-12 h-12" />}
          title="No events found"
          description="Try adjusting your search or filters."
        />
      ) : (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {events.map((event) => (
              <EventCard key={event.id} event={event} />
            ))}
          </div>

          {/* Pagination */}
          <div className="flex items-center justify-center gap-4 pt-4">
            <Button
              variant="ghost"
              size="sm"
              disabled={page === 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
            >
              <ChevronLeft className="w-4 h-4" /> Previous
            </Button>
            <span className="text-sm text-nu-text-muted">Page {page + 1}</span>
            <Button
              variant="ghost"
              size="sm"
              disabled={!hasMore}
              onClick={() => setPage((p) => p + 1)}
            >
              Next <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </>
      )}
    </div>
  );
}
