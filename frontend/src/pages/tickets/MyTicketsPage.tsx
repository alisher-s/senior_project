import { useState } from 'react';
import { useTicketsStore } from '../../stores/tickets';
import { useToastStore } from '../../stores/toast';
import { ticketsAPI } from '../../api/services';
import { getErrorCode } from '../../api/client';
import { Button, Badge, EmptyState, Modal } from '../../components/ui/Primitives';
import { formatEventDate, isEventPast } from '../../lib/utils';
import { Ticket, CalendarDays, QrCode, X, Maximize2 } from 'lucide-react';
import type { StoredTicket } from '../../types';

export default function MyTicketsPage() {
  const { tickets, updateStatus } = useTicketsStore();
  const toast = useToastStore();
  const [selectedTicket, setSelectedTicket] = useState<StoredTicket | null>(null);
  const [fullscreen, setFullscreen] = useState(false);
  const [cancelling, setCancelling] = useState<string | null>(null);

  const activeTickets = tickets.filter((t) => t.status === 'active');
  const pastTickets = tickets.filter((t) => t.status !== 'active');

  const handleCancel = async (ticket: StoredTicket) => {
    setCancelling(ticket.ticket_id);
    try {
      await ticketsAPI.cancel(ticket.ticket_id);
      updateStatus(ticket.ticket_id, 'cancelled');
      toast.add('Ticket cancelled.', 'info');
    } catch (err) {
      const code = getErrorCode(err);
      const messages: Record<string, string> = {
        ticket_not_found: 'Ticket not found.',
        ticket_already_cancelled: 'Already cancelled.',
        ticket_already_used: 'This ticket has been used.',
        cancellation_not_allowed: 'Cannot cancel after event start.',
      };
      toast.add(messages[code] || 'Failed to cancel ticket.', 'error');
    } finally {
      setCancelling(null);
    }
  };

  if (tickets.length === 0) {
    return (
      <EmptyState
        icon={<Ticket className="w-12 h-12" />}
        title="No tickets yet"
        description="Browse events and register to get your tickets."
      />
    );
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="font-display text-2xl font-bold">My Tickets</h1>
        <p className="text-nu-text-muted text-sm mt-1">Your registered events and QR codes</p>
      </div>

      {/* Active tickets */}
      {activeTickets.length > 0 && (
        <div className="space-y-3">
          <h2 className="font-display font-semibold text-lg">Active</h2>
          {activeTickets.map((ticket) => (
            <TicketRow
              key={ticket.ticket_id}
              ticket={ticket}
              onView={() => setSelectedTicket(ticket)}
              onCancel={() => handleCancel(ticket)}
              cancelling={cancelling === ticket.ticket_id}
            />
          ))}
        </div>
      )}

      {/* Past / cancelled tickets */}
      {pastTickets.length > 0 && (
        <div className="space-y-3">
          <h2 className="font-display font-semibold text-lg text-nu-text-muted">Past & Cancelled</h2>
          {pastTickets.map((ticket) => (
            <TicketRow
              key={ticket.ticket_id}
              ticket={ticket}
              onView={() => setSelectedTicket(ticket)}
              disabled
            />
          ))}
        </div>
      )}

      {/* QR Modal */}
      <Modal
        open={!!selectedTicket}
        onClose={() => { setSelectedTicket(null); setFullscreen(false); }}
        title="Your Ticket"
      >
        {selectedTicket && (
          <div className="flex flex-col items-center gap-4">
            <h3 className="font-display font-bold text-center">{selectedTicket.event_title}</h3>
            <p className="text-sm text-nu-text-muted">
              {formatEventDate(selectedTicket.event_starts_at)}
            </p>

            {/* QR Code */}
            <div className="relative bg-white rounded-2xl p-4">
              <img
                src={`data:image/png;base64,${selectedTicket.qr_png_base64}`}
                alt="Ticket QR Code"
                className={fullscreen ? 'w-80 h-80' : 'w-48 h-48'}
              />
              <button
                onClick={() => setFullscreen(!fullscreen)}
                className="absolute top-2 right-2 p-1 rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600"
              >
                <Maximize2 className="w-4 h-4" />
              </button>
            </div>

            <Badge variant={selectedTicket.status === 'active' ? 'success' : 'default'}>
              {selectedTicket.status}
            </Badge>

            <p className="text-xs text-nu-text-muted text-center">
              Show this QR code at the event entrance for check-in
            </p>

            <Button variant="ghost" onClick={() => { setSelectedTicket(null); setFullscreen(false); }}>
              Close
            </Button>
          </div>
        )}
      </Modal>
    </div>
  );
}

function TicketRow({
  ticket,
  onView,
  onCancel,
  cancelling = false,
  disabled = false,
}: {
  ticket: StoredTicket;
  onView: () => void;
  onCancel?: () => void;
  cancelling?: boolean;
  disabled?: boolean;
}) {
  const past = isEventPast(ticket.event_starts_at);

  return (
    <div className="flex items-center gap-4 rounded-xl border border-white/5 bg-nu-surface/50 p-4">
      <button
        onClick={onView}
        className="shrink-0 w-14 h-14 rounded-xl bg-white flex items-center justify-center overflow-hidden"
      >
        {ticket.qr_png_base64 ? (
          <img
            src={`data:image/png;base64,${ticket.qr_png_base64}`}
            alt="QR"
            className="w-12 h-12"
          />
        ) : (
          <QrCode className="w-6 h-6 text-gray-400" />
        )}
      </button>

      <div className="flex-1 min-w-0">
        <h3 className="font-semibold text-sm truncate">{ticket.event_title}</h3>
        <div className="flex items-center gap-2 text-xs text-nu-text-muted mt-1">
          <CalendarDays className="w-3 h-3" />
          {formatEventDate(ticket.event_starts_at)}
        </div>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <Badge
          variant={
            ticket.status === 'active' ? (past ? 'warning' : 'success') :
            ticket.status === 'used' ? 'gold' : 'error'
          }
        >
          {ticket.status === 'active' && past ? 'past' : ticket.status}
        </Badge>

        <Button variant="secondary" size="sm" onClick={onView}>
          <QrCode className="w-3.5 h-3.5" />
        </Button>

        {onCancel && !disabled && ticket.status === 'active' && !past && (
          <Button variant="danger" size="sm" onClick={onCancel} loading={cancelling}>
            <X className="w-3.5 h-3.5" />
          </Button>
        )}
      </div>
    </div>
  );
}
