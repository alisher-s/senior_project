import api from './client';
import type {
  AuthResponse,
  RegisterRequest,
  LoginRequest,
  EventDTO,
  ListEventsResponse,
  CreateEventRequest,
  UpdateEventRequest,
  RegisterTicketResponse,
  CancelTicketResponse,
  UseTicketResponse,
  InitiatePaymentRequest,
  InitiatePaymentResponse,
  ModerateEventRequest,
  ModerateEventResponse,
  UserRoleResponse,
  SetUserRoleRequest,
  EventStatsResponse,
} from '../types';

// ─── Auth ─────────────────────────────────────────
export const authAPI = {
  register: (data: RegisterRequest) =>
    api.post<AuthResponse>('/auth/register', data),

  login: (data: LoginRequest) =>
    api.post<AuthResponse>('/auth/login', data),

  refresh: (refresh_token: string) =>
    api.post<AuthResponse>('/auth/refresh', { refresh_token }),
};

// ─── Events ───────────────────────────────────────
export interface EventFilters {
  q?: string;
  limit?: number;
  offset?: number;
  starts_after?: string;
  starts_before?: string;
}

export const eventsAPI = {
  list: (filters?: EventFilters) =>
    api.get<ListEventsResponse>('/events/', { params: filters }),

  getById: (id: string) =>
    api.get<EventDTO>(`/events/${id}`),

  create: (data: CreateEventRequest) =>
    api.post<EventDTO>('/events/', data),

  update: (id: string, data: UpdateEventRequest) =>
    api.put<EventDTO>(`/events/${id}`, data),

  delete: (id: string) =>
    api.delete(`/events/${id}`),
};

// ─── Tickets ──────────────────────────────────────
export const ticketsAPI = {
  register: (event_id: string) =>
    api.post<RegisterTicketResponse>('/tickets/register', { event_id }),

  cancel: (ticketId: string) =>
    api.post<CancelTicketResponse>(`/tickets/${ticketId}/cancel`),

  use: (qr_hash_hex: string) =>
    api.post<UseTicketResponse>('/tickets/use', { qr_hash_hex }),
};

// ─── Payments ─────────────────────────────────────
export const paymentsAPI = {
  initiate: (data: InitiatePaymentRequest) =>
    api.post<InitiatePaymentResponse>('/payments/initiate', data),
};

// ─── Admin ────────────────────────────────────────
export const adminAPI = {
  moderateEvent: (eventId: string, data: ModerateEventRequest) =>
    api.post<ModerateEventResponse>(`/admin/events/${eventId}/moderate`, data),

  setUserRole: (userId: string, data: SetUserRoleRequest) =>
    api.patch<UserRoleResponse>(`/admin/users/${userId}/role`, data),
};

// ─── Analytics ────────────────────────────────────
export const analyticsAPI = {
  eventStats: (eventId?: string) =>
    api.get<EventStatsResponse>('/analytics/events/stats', {
      params: eventId ? { event_id: eventId } : undefined,
    }),
};
