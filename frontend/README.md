# NU Events — Frontend

React + TypeScript + Vite frontend for the Student Event Ticketing Platform at Nazarbayev University.

Connects to the Go backend at `http://localhost:8080/api/v1`.

## Quick Start

```bash
# 1. Make sure the backend is running
cd ../senior_project
docker compose up --build

# 2. Install frontend dependencies
npm install

# 3. Start dev server (port 5173, proxies /api to backend)
npm run dev
```

Open `http://localhost:5173` in your browser.

## Test Accounts (from backend migrations)

| Role      | Email                        | Password        |
|-----------|------------------------------|-----------------|
| Organizer | staff.organizer@nu.edu.kz    | DevStaffPass1!  |
| Admin     | staff.admin@nu.edu.kz        | DevStaffPass1!  |
| Student   | (register any @nu.edu.kz)    | (min 8 chars)   |

## Project Structure

```
src/
├── api/
│   ├── client.ts          # Axios instance, JWT interceptors, auto-refresh
│   └── services.ts        # All API calls matching backend endpoints
├── components/
│   ├── auth/Guards.tsx     # ProtectedRoute, RoleGuard
│   ├── events/
│   │   ├── EventCard.tsx   # Card + skeleton for catalog grid
│   │   └── EventFilters.tsx # Search + date filter bar
│   ├── layout/AppShell.tsx # Header, nav, footer, role-aware menu
│   └── ui/
│       ├── Primitives.tsx  # Button, Input, Textarea, Modal, Badge, Spinner, etc.
│       └── Toaster.tsx     # Toast notification system
├── lib/utils.ts            # Date formatting, capacity helpers
├── pages/
│   ├── HomePage.tsx
│   ├── auth/LoginPage.tsx, RegisterPage.tsx
│   ├── events/EventsPage.tsx, EventDetailPage.tsx
│   ├── tickets/MyTicketsPage.tsx
│   ├── organizer/OrganizerDashboard.tsx, CreateEventPage.tsx, EditEventPage.tsx, CheckInPage.tsx
│   └── admin/AdminPanel.tsx
├── stores/auth.ts, tickets.ts, toast.ts
├── types/index.ts
├── App.tsx
├── main.tsx
└── index.css
```

## Backend Endpoints Used

| Method | Path | Page |
|--------|------|------|
| POST | /auth/register | RegisterPage |
| POST | /auth/login | LoginPage |
| POST | /auth/refresh | (auto, client.ts) |
| GET | /events/ | EventsPage, HomePage |
| GET | /events/:id | EventDetailPage |
| POST | /events/ | CreateEventPage |
| PUT | /events/:id | EditEventPage |
| DELETE | /events/:id | EditEventPage |
| POST | /tickets/register | EventDetailPage |
| POST | /tickets/:id/cancel | MyTicketsPage |
| POST | /tickets/use | CheckInPage |
| POST | /payments/initiate | (wired, backend stub) |
| POST | /admin/events/:id/moderate | AdminPanel |
| PATCH | /admin/users/:id/role | AdminPanel |

## Stack

React 19, TypeScript, Vite 6, Tailwind CSS v4, TanStack React Query, Zustand, React Router v7, Axios, Lucide React, date-fns.

## Build

```bash
npm run build    # outputs to dist/
npm run preview  # preview production build locally
```
