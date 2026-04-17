import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { eventsAPI } from '../api/services';
import { EventCard, EventCardSkeleton } from '../components/events/EventCard';
import { Button } from '../components/ui/Primitives';
import { ArrowRight, CalendarDays, Ticket, Shield } from 'lucide-react';
import { useAuthStore } from '../stores/auth';

export default function HomePage() {
  const { isAuthenticated } = useAuthStore();

  const { data, isLoading } = useQuery({
    queryKey: ['events', 'featured'],
    queryFn: () =>
      eventsAPI
        .list({
          limit: 6,
          starts_after: new Date().toISOString(),
        })
        .then((r) => r.data),
  });

  const events = data?.items || [];

  return (
    <div className="space-y-16">
      {/* Hero */}
      <section className="text-center py-12 sm:py-20">
        <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-nu-gold/10 text-nu-gold text-sm font-medium mb-6">
          <CalendarDays className="w-4 h-4" />
          Nazarbayev University Events
        </div>
        <h1 className="font-display text-4xl sm:text-5xl lg:text-6xl font-bold leading-tight max-w-3xl mx-auto">
          Discover &amp; Register for
          <span className="text-nu-gold"> Campus Events</span>
        </h1>
        <p className="text-nu-text-muted text-lg mt-4 max-w-xl mx-auto">
          Browse upcoming events, register with one click, and get your digital ticket with QR code for instant check-in.
        </p>
        <div className="flex items-center justify-center gap-4 mt-8">
          <Link to="/events">
            <Button size="lg">
              Browse Events <ArrowRight className="w-4 h-4" />
            </Button>
          </Link>
          {!isAuthenticated && (
            <Link to="/register">
              <Button variant="secondary" size="lg">
                Create Account
              </Button>
            </Link>
          )}
        </div>
      </section>

      {/* Features */}
      <section className="grid grid-cols-1 sm:grid-cols-3 gap-6">
        <FeatureCard
          icon={<CalendarDays className="w-6 h-6" />}
          title="Event Catalog"
          description="Browse all approved events with search, date filtering, and real-time capacity tracking."
        />
        <FeatureCard
          icon={<Ticket className="w-6 h-6" />}
          title="Digital Tickets"
          description="Register instantly and receive a QR code ticket for seamless check-in at the venue."
        />
        <FeatureCard
          icon={<Shield className="w-6 h-6" />}
          title="Secure & Verified"
          description="NU email authentication, one-ticket-per-student enforcement, and admin moderation."
        />
      </section>

      {/* Upcoming events */}
      <section>
        <div className="flex items-center justify-between mb-6">
          <h2 className="font-display text-xl font-bold">Upcoming Events</h2>
          <Link to="/events" className="text-sm text-nu-gold hover:text-nu-gold-light font-medium flex items-center gap-1">
            View all <ArrowRight className="w-3.5 h-3.5" />
          </Link>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <EventCardSkeleton key={i} />
            ))}
          </div>
        ) : events.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {events.map((event) => (
              <EventCard key={event.id} event={event} />
            ))}
          </div>
        ) : (
          <p className="text-center text-nu-text-muted py-8">No upcoming events right now. Check back soon!</p>
        )}
      </section>
    </div>
  );
}

function FeatureCard({ icon, title, description }: { icon: React.ReactNode; title: string; description: string }) {
  return (
    <div className="rounded-2xl border border-white/5 bg-nu-surface/30 p-6 hover:border-nu-gold/20 transition-all">
      <div className="text-nu-gold mb-4">{icon}</div>
      <h3 className="font-display font-bold mb-2">{title}</h3>
      <p className="text-sm text-nu-text-muted leading-relaxed">{description}</p>
    </div>
  );
}
