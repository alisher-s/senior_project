import { useState, useEffect } from 'react';
import { Search, X } from 'lucide-react';

interface EventFiltersProps {
  onFilterChange: (filters: {
    q?: string;
    starts_after?: string;
    starts_before?: string;
  }) => void;
}

export default function EventFilters({ onFilterChange }: EventFiltersProps) {
  const [query, setQuery] = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => {
      onFilterChange({
        q: query || undefined,
        starts_after: dateFrom ? new Date(dateFrom).toISOString() : undefined,
        starts_before: dateTo ? new Date(dateTo).toISOString() : undefined,
      });
    }, 300);
    return () => clearTimeout(timer);
  }, [query, dateFrom, dateTo]);

  const clearAll = () => {
    setQuery('');
    setDateFrom('');
    setDateTo('');
  };

  const hasFilters = query || dateFrom || dateTo;

  return (
    <div className="flex flex-col sm:flex-row gap-3">
      {/* Search input */}
      <div className="relative flex-1">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-nu-text-muted" />
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search events..."
          className="w-full rounded-xl border border-white/10 bg-nu-surface pl-10 pr-4 py-2.5 text-sm text-nu-text placeholder:text-nu-text-muted/50 focus:outline-none focus:ring-2 focus:ring-nu-gold/50 transition-all"
        />
      </div>

      {/* Date filters */}
      <div className="flex gap-2">
        <input
          type="date"
          value={dateFrom}
          onChange={(e) => setDateFrom(e.target.value)}
          className="rounded-xl border border-white/10 bg-nu-surface px-3 py-2.5 text-sm text-nu-text focus:outline-none focus:ring-2 focus:ring-nu-gold/50 transition-all"
          title="From date"
        />
        <input
          type="date"
          value={dateTo}
          onChange={(e) => setDateTo(e.target.value)}
          className="rounded-xl border border-white/10 bg-nu-surface px-3 py-2.5 text-sm text-nu-text focus:outline-none focus:ring-2 focus:ring-nu-gold/50 transition-all"
          title="To date"
        />
        {hasFilters && (
          <button
            onClick={clearAll}
            className="rounded-xl border border-white/10 bg-nu-surface px-3 py-2.5 text-sm text-nu-text-muted hover:text-nu-text hover:bg-white/5 transition-all"
            title="Clear filters"
          >
            <X className="w-4 h-4" />
          </button>
        )}
      </div>
    </div>
  );
}
