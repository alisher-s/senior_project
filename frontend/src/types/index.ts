// Auth
export interface UserDTO {
  id: string;
  email: string;
  role: 'student' | 'organizer' | 'admin';
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: UserDTO;
}

export interface RegisterRequest {
  email: string;
  password: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

// Events
export interface EventDTO {
  id: string;
  title: string;
  description: string;
  starts_at: string;
  capacity_total: number;
  capacity_available: number;
  status: 'draft' | 'published' | 'cancelled';
  moderation_status: 'pending' | 'approved' | 'rejected';
}

export interface ListEventsResponse {
  items: EventDTO[];
  limit: number;
  offset: number;
}

export interface CreateEventRequest {
  title: string;
  description: string;
  starts_at: string;
  capacity_total: number;
}

export interface UpdateEventRequest {
  title?: string;
  description?: string;
  starts_at?: string;
  capacity_total?: number;
  status?: 'draft' | 'published' | 'cancelled';
}

// Tickets
export interface RegisterTicketRequest {
  event_id: string;
}

export interface RegisterTicketResponse {
  ticket_id: string;
  event_id: string;
  user_id: string;
  status: string;
  qr_png_base64: string;
  qr_hash_hex: string;
}

export interface CancelTicketResponse {
  ticket_id: string;
  event_id: string;
  user_id: string;
  status: string;
}

export interface UseTicketRequest {
  qr_hash_hex: string;
}

export interface UseTicketResponse {
  ticket_id: string;
  event_id: string;
  user_id: string;
  status: string;
}

// Payments
export interface InitiatePaymentRequest {
  event_id: string;
  amount: number;
  currency: string;
}

export interface InitiatePaymentResponse {
  payment_id: string;
  provider_ref: string;
  provider_url: string;
}

// Admin
export interface ModerateEventRequest {
  action: 'approve' | 'reject';
  reason?: string;
}

export interface ModerateEventResponse {
  moderation_status: string;
}

export interface SetUserRoleRequest {
  role: 'student' | 'organizer' | 'admin';
}

export interface UserRoleResponse {
  id: string;
  email: string;
  role: string;
}

// Analytics
export interface EventStatsResponse {
  event_id: string;
  tickets: number;
  revenue: number;
  as_of: string;
}

// API Error
export interface APIError {
  error: {
    code: string;
    message: string;
  };
}

// Local ticket type (stored client-side after registration)
export interface StoredTicket {
  ticket_id: string;
  event_id: string;
  event_title: string;
  event_starts_at: string;
  status: string;
  qr_png_base64: string;
  qr_hash_hex: string;
  registered_at: string;
}
