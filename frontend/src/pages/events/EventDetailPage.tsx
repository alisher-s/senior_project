import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { eventsAPI, ticketsAPI } from '../../api/services';
import { useAuthStore } from '../../stores/auth';
import { useTicketsStore } from '../../stores/tickets';
import { useToastStore } from '../../stores/toast';
import { getErrorCode, getErrorMessage } from '../../api/client';
import { Button, Badge, Spinner, Modal } from '../../components/ui/Primitives';
import { formatEventDate, formatRelative, capacityPercent, capacityColor, isEventPast } from '../../lib/utils';
import { CalendarDays, Users, ArrowLeft, Ticket, CheckCircle } from 'lucide-react';

export default function EventDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user, isAuthenticated } = useAuthStore();
  const { cacheQR } = useTicketsStore();
  const toast = useToastStore();
  const [registering, setRegistering] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const { data: event, isLoading, isError } = useQuery({
    queryKey: ['event', id],
    queryFn: () => eventsAPI.getById(id!).then((r) => r.data),
    enabled: !!id,
  });

  // Check if user already has a ticket for this event via server
  const { data: myTickets } = useQuery({
    queryKey: ['my-tickets'],
    queryFn: () => ticketsAPI.my().then((r) => r.data),
    enabled: isAuthenticated,
  });

  const existingTicket = myTickets?.tickets.find(
    (t) => t.event_id === id && (t.status === 'active' || t.status === 'used')
  );

  const isFull = event ? event.capacity_available <= 0 : false;
  const past = event ? isEventPast(event.starts_at) : false;

  const handleRegister = async () => {
    if (!event || !id) return;
    setConfirmOpen(false);
    setRegistering(true);

    try {
      const { data } = await ticketsAPI.register(id);
      // Cache QR locally since GET /tickets/my doesn't return it
      cacheQR(data.ticket_id, data.qr_png_base64);
      toast.add('Registered! Your ticket is ready.', 'success');
      queryClient.invalidateQueries({ queryKey: ['my-tickets'] });
      queryClient.invalidateQueries({ queryKey: ['event', id] });
      navigate(`/my/tickets`);
    } catch (err) {
      const code = getErrorCode(err);
      const messages: Record<string, string> = {
        already_registered: "You're already registered for this event.",
        capacity_full: 'This event is full.',
        event_not_approved: 'This event is not yet approved.',
        event_not_published: 'This event is not open for registration.',
        event_cancelled: 'This event has been cancelled.',
        registration_closed: 'Registration is closed for this event.',
      };
      toast.add(messages[code] || getErrorMessage(err), 'error');
    } finally {
      setRegistering(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Spinner className="w-8 h-8" />
      </div>
    );
  }

  if (isError || !event) {
    return (
      <div className="text-center py-20">
        <h2 className="font-display text-xl font-bold mb-2">Event not found</h2>
        <p className="text-nu-text-muted text-sm mb-4">This event may have been removed or is not yet approved.</p>
        <Button variant="secondary" onClick={() => navigate('/events')}>
          <ArrowLeft className="w-4 h-4" /> Back to Events
        </Button>
      </div>
    );
  }

  const pct = capacityPercent(event.capacity_total, event.capacity_available);

  return (
    <div className="max-w-3xl mx-auto">
      <button
        onClick={() => navigate('/events')}
        className="flex items-center gap-2 text-sm text-nu-text-muted hover:text-nu-text mb-6 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" /> Back to Events
      </button>

      <div className="rounded-2xl border border-white/5 bg-nu-surface/50 overflow-hidden">
        {/* Cover image or accent bar */}
        {event.cover_image_url ? (
          <div className="h-48 sm:h-64 overflow-hidden">
            <img
              src={event.cover_image_url}
              alt={event.title}
              className="w-full h-full object-cover"
            />
          </div>
        ) : (
          <div className="h-2 bg-gradient-to-r from-nu-gold to-nu-gold-light" />
        )}

        <div className="p-6 sm:p-8">
          <div className="flex flex-wrap gap-2 mb-4">
            <Badge variant={event.status === 'published' ? 'success' : event.status === 'cancelled' ? 'error' : 'warning'}>
              {event.status}
            </Badge>
            {past && <Badge variant="error">Past Event</Badge>}
          </div>

          <h1 className="font-display text-2xl sm:text-3xl font-bold mb-4">{event.title}</h1>

          <div className="flex flex-col gap-3 mb-6 text-nu-text-muted">
            <div className="flex items-center gap-3">
              <CalendarDays className="w-5 h-5 text-nu-gold" />
              <div>
                <p className="text-nu-text">{formatEventDate(event.starts_at)}</p>
                <p className="text-xs">{formatRelative(event.starts_at)}</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Users className="w-5 h-5 text-nu-gold" />
              <div>
                <p className={capacityColor(event.capacity_total, event.capacity_available)}>
                  {event.capacity_available} spots remaining
                </p>
                <p className="text-xs">
                  {event.capacity_total - event.capacity_available} / {event.capacity_total} registered
                </p>
              </div>
            </div>
          </div>

          {/* Capacity bar */}
          <div className="h-2 rounded-full bg-white/5 overflow-hidden mb-8">
            <div
              className="h-full rounded-full bg-gradient-to-r from-nu-gold to-nu-gold-light transition-all duration-500"
              style={{ width: `${Math.min(pct, 100)}%` }}
            />
          </div>

          {/* Description */}
          {event.description && (
            <div className="mb-8">
              <h2 className="font-display font-semibold mb-2">About</h2>
              <p className="text-nu-text-muted leading-relaxed whitespace-pre-wrap">{event.description}</p>
            </div>
          )}

          {/* Action area */}
          <div className="border-t border-white/5 pt-6">
            {existingTicket ? (
              <div className="flex items-center gap-3">
                <CheckCircle className="w-5 h-5 text-nu-success" />
                <span className="text-nu-success font-medium">You're registered!</span>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => navigate('/my/tickets')}
                >
                  <Ticket className="w-4 h-4" /> View Ticket
                </Button>
              </div>
            ) : !isAuthenticated ? (
              <Button onClick={() => navigate('/login', { state: { from: `/events/${id}` } })} size="lg">
                Sign in to Register
              </Button>
            ) : user?.role !== 'student' ? (
              <p className="text-nu-text-muted text-sm">Only students can register for events.</p>
            ) : past ? (
              <Button disabled size="lg">Event has ended</Button>
            ) : isFull ? (
              <Button disabled size="lg">Event Full</Button>
            ) : event.status === 'cancelled' ? (
              <Button disabled size="lg">Event Cancelled</Button>
            ) : (
              <Button onClick={() => setConfirmOpen(true)} loading={registering} size="lg">
                <Ticket className="w-4 h-4" /> Register for Event
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Confirmation modal */}
      <Modal open={confirmOpen} onClose={() => setConfirmOpen(false)} title="Confirm Registration">
        <p className="text-sm text-nu-text-muted mb-6">
          You're about to register for <strong className="text-nu-text">{event.title}</strong>.
          You'll receive a QR code ticket for check-in.
        </p>
        <div className="flex gap-3 justify-end">
          <Button variant="ghost" onClick={() => setConfirmOpen(false)}>Cancel</Button>
          <Button onClick={handleRegister} loading={registering}>Confirm</Button>
        </div>
      </Modal>
    </div>
  );
}
