// Auth
export interface UserDTO {
  id: string;
  email: string;
  role: 'student' | 'organizer' | 'admin';
  roles: string[];
  pending_roles?: string[];
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
  cover_image_url?: string;
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
  cover_image_url?: string;
  starts_at: string;
  capacity_total: number;
}

export interface UpdateEventRequest {
  title?: string;
  description?: string;
  cover_image_url?: string;
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

// GET /tickets/my response
export interface MyTicketsResponse {
  tickets: MyTicketItem[];
}

export interface MyTicketItem {
  ticket_id: string;
  status: 'active' | 'used' | 'cancelled' | 'expired';
  qr_hash_hex: string;
  event_id: string;
  event_title: string;
  event_date: string;
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

export interface ModerationLogEntry {
  id: string;
  admin_user_id: string;
  event_id?: string;
  action: string;
  reason?: string;
  created_at: string;
}

export interface ModerationLogsResponse {
  items: ModerationLogEntry[];
  limit: number;
  offset: number;
}

// Analytics (now fully implemented)
export interface RegistrationHour {
  hour: string;
  count: number;
}

export interface EventStatsResponse {
  event_id?: string;
  total_capacity: number;
  registered_count: number;
  remaining_capacity: number;
  registration_timeline: RegistrationHour[];
  as_of: string;
}

// API Error
export interface APIError {
  error: {
    code: string;
    message: string;
  };
}
