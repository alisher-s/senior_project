import { create } from 'zustand';

// QR images are cached client-side since GET /tickets/my doesn't return them.
// The server is now the source of truth for ticket list/status.

interface QRCache {
  [ticketId: string]: string; // ticket_id -> qr_png_base64
}

interface TicketsState {
  qrCache: QRCache;
  cacheQR: (ticketId: string, qrBase64: string) => void;
  getQR: (ticketId: string) => string | undefined;
  hydrate: () => void;
}

const STORAGE_KEY = 'nu_qr_cache';

export const useTicketsStore = create<TicketsState>((set, get) => ({
  qrCache: {},

  cacheQR: (ticketId, qrBase64) => {
    const next = { ...get().qrCache, [ticketId]: qrBase64 };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    set({ qrCache: next });
  },

  getQR: (ticketId) => get().qrCache[ticketId],

  hydrate: () => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) set({ qrCache: JSON.parse(stored) });
    } catch {
      localStorage.removeItem(STORAGE_KEY);
    }
  },
}));
