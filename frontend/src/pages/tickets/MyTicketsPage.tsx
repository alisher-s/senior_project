import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ticketsAPI } from '../../api/services';
import { useTicketsStore } from '../../stores/tickets';
import { useToastStore } from '../../stores/toast';
import { getErrorCode } from '../../api/client';
import { Button, Badge, EmptyState, Modal, Spinner } from '../../components/ui/Primitives';
import { formatEventDate } from '../../lib/utils';
import { Ticket, CalendarDays, QrCode, X, Maximize2 } from 'lucide-react';
import type { MyTicketItem } from '../../types';

export default function MyTicketsPage() {
  const queryClient = useQueryClient();
  const { getQR } = useTicketsStore();
  const toast = useToastStore();
  const [selectedTicket, setSelectedTicket] = useState<MyTicketItem | null>(null);
  const [fullscreen, setFullscreen] = useState(false);
  const [cancelling, setCancelling] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['my-tickets'],
    queryFn: () => ticketsAPI.my().then((r) => r.data),
  });

  const tickets = data?.tickets || [];
  const activeTickets = tickets.filter((t) => t.status === 'active');
  const otherTickets = tickets.filter((t) => t.status !== 'active');

  const handleCancel = async (ticket: MyTicketItem) => {
    setCancelling(ticket.ticket_id);
    try {
      await ticketsAPI.cancel(ticket.ticket_id);
      toast.add('Ticket cancelled.', 'info');
      queryClient.invalidateQueries({ queryKey: ['my-tickets'] });
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

  if (isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Spinner className="w-8 h-8" />
      </div>
    );
  }

  if (tickets.length === 0) {
    return (
      <EmptyState
        icon={<Ticket className="w-12 h-12" />}
        title="No tickets yet"
        description="Browse events and register to get your tickets."
      />
    );
  }

  const qrForTicket = selectedTicket ? getQR(selectedTicket.ticket_id) : undefined;

  return (
    <div className="space-y-8">
      <div>
        <h1 className="font-display text-2xl font-bold">My Tickets</h1>
        <p className="text-nu-text-muted text-sm mt-1">Your registered events and QR codes</p>
      </div>

      {activeTickets.length > 0 && (
        <div className="space-y-3">
          <h2 className="font-display font-semibold text-lg">Active</h2>
          {activeTickets.map((ticket) => (
            <TicketRow
              key={ticket.ticket_id}
              ticket={ticket}
              qr={getQR(ticket.ticket_id)}
              onView={() => setSelectedTicket(ticket)}
              onCancel={() => handleCancel(ticket)}
              cancelling={cancelling === ticket.ticket_id}
            />
          ))}
        </div>
      )}

      {otherTickets.length > 0 && (
        <div className="space-y-3">
          <h2 className="font-display font-semibold text-lg text-nu-text-muted">Past & Other</h2>
          {otherTickets.map((ticket) => (
            <TicketRow
              key={ticket.ticket_id}
              ticket={ticket}
              qr={getQR(ticket.ticket_id)}
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
              {formatEventDate(selectedTicket.event_date)}
            </p>

            {qrForTicket ? (
              <div className="relative bg-white rounded-2xl p-4">
                <img
                  src={`data:image/png;base64,${qrForTicket}`}
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
            ) : (
              <div className="bg-nu-surface rounded-2xl p-8 text-center">
                <QrCode className="w-12 h-12 text-nu-text-muted mx-auto mb-2" />
                <p className="text-sm text-nu-text-muted">QR code not cached locally.</p>
                <p className="text-xs text-nu-text-muted mt-1">Hash: {selectedTicket.qr_hash_hex}</p>
              </div>
            )}

            <Badge variant={selectedTicket.status === 'active' ? 'success' : selectedTicket.status === 'expired' ? 'warning' : 'default'}>
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
  qr,
  onView,
  onCancel,
  cancelling = false,
  disabled = false,
}: {
  ticket: MyTicketItem;
  qr?: string;
  onView: () => void;
  onCancel?: () => void;
  cancelling?: boolean;
  disabled?: boolean;
}) {
  const statusVariant = {
    active: 'success' as const,
    used: 'gold' as const,
    cancelled: 'error' as const,
    expired: 'warning' as const,
  };

  return (
    <div className="flex items-center gap-4 rounded-xl border border-white/5 bg-nu-surface/50 p-4">
      <button
        onClick={onView}
        className="shrink-0 w-14 h-14 rounded-xl bg-white flex items-center justify-center overflow-hidden"
      >
        {qr ? (
          <img src={`data:image/png;base64,${qr}`} alt="QR" className="w-12 h-12" />
        ) : (
          <QrCode className="w-6 h-6 text-gray-400" />
        )}
      </button>

      <div className="flex-1 min-w-0">
        <h3 className="font-semibold text-sm truncate">{ticket.event_title}</h3>
        <div className="flex items-center gap-2 text-xs text-nu-text-muted mt-1">
          <CalendarDays className="w-3 h-3" />
          {formatEventDate(ticket.event_date)}
        </div>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <Badge variant={statusVariant[ticket.status] || 'default'}>
          {ticket.status}
        </Badge>

        <Button variant="secondary" size="sm" onClick={onView}>
          <QrCode className="w-3.5 h-3.5" />
        </Button>

        {onCancel && !disabled && ticket.status === 'active' && (
          <Button variant="danger" size="sm" onClick={onCancel} loading={cancelling}>
            <X className="w-3.5 h-3.5" />
          </Button>
        )}
      </div>
    </div>
  );
}
