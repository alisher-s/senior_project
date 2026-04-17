import { create } from 'zustand';
import type { StoredTicket } from '../types';

interface TicketsState {
  tickets: StoredTicket[];
  addTicket: (ticket: StoredTicket) => void;
  removeTicket: (ticketId: string) => void;
  updateStatus: (ticketId: string, status: string) => void;
  getByEvent: (eventId: string) => StoredTicket | undefined;
  hydrate: () => void;
}

const STORAGE_KEY = 'nu_tickets';

export const useTicketsStore = create<TicketsState>((set, get) => ({
  tickets: [],

  addTicket: (ticket) => {
    const current = get().tickets.filter((t) => t.ticket_id !== ticket.ticket_id);
    const next = [ticket, ...current];
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    set({ tickets: next });
  },

  removeTicket: (ticketId) => {
    const next = get().tickets.filter((t) => t.ticket_id !== ticketId);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    set({ tickets: next });
  },

  updateStatus: (ticketId, status) => {
    const next = get().tickets.map((t) =>
      t.ticket_id === ticketId ? { ...t, status } : t
    );
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    set({ tickets: next });
  },

  getByEvent: (eventId) => {
    return get().tickets.find(
      (t) => t.event_id === eventId && t.status === 'active'
    );
  },

  hydrate: () => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) set({ tickets: JSON.parse(stored) });
    } catch {
      localStorage.removeItem(STORAGE_KEY);
    }
  },
}));
