import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import AppShell from './components/layout/AppShell';
import { ProtectedRoute, RoleGuard } from './components/auth/Guards';
import Toaster from './components/ui/Toaster';

// Pages
import HomePage from './pages/HomePage';
import LoginPage from './pages/auth/LoginPage';
import RegisterPage from './pages/auth/RegisterPage';
import EventsPage from './pages/events/EventsPage';
import EventDetailPage from './pages/events/EventDetailPage';
import MyTicketsPage from './pages/tickets/MyTicketsPage';
import OrganizerDashboard from './pages/organizer/OrganizerDashboard';
import CreateEventPage from './pages/organizer/CreateEventPage';
import EditEventPage from './pages/organizer/EditEventPage';
import CheckInPage from './pages/organizer/CheckInPage';
import AdminPanel from './pages/admin/AdminPanel';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<AppShell />}>
            {/* Public */}
            <Route path="/" element={<HomePage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/events" element={<EventsPage />} />
            <Route path="/events/:id" element={<EventDetailPage />} />

            {/* Student (any authenticated user) */}
            <Route
              path="/my/tickets"
              element={
                <ProtectedRoute>
                  <MyTicketsPage />
                </ProtectedRoute>
              }
            />

            {/* Organizer */}
            <Route
              path="/organizer"
              element={
                <ProtectedRoute>
                  <RoleGuard allow={['organizer', 'admin']}>
                    <OrganizerDashboard />
                  </RoleGuard>
                </ProtectedRoute>
              }
            />
            <Route
              path="/organizer/events/new"
              element={
                <ProtectedRoute>
                  <RoleGuard allow={['organizer', 'admin']}>
                    <CreateEventPage />
                  </RoleGuard>
                </ProtectedRoute>
              }
            />
            <Route
              path="/organizer/events/:id/edit"
              element={
                <ProtectedRoute>
                  <RoleGuard allow={['organizer', 'admin']}>
                    <EditEventPage />
                  </RoleGuard>
                </ProtectedRoute>
              }
            />
            <Route
              path="/organizer/check-in"
              element={
                <ProtectedRoute>
                  <RoleGuard allow={['organizer', 'admin']}>
                    <CheckInPage />
                  </RoleGuard>
                </ProtectedRoute>
              }
            />

            {/* Admin */}
            <Route
              path="/admin"
              element={
                <ProtectedRoute>
                  <RoleGuard allow={['admin']}>
                    <AdminPanel />
                  </RoleGuard>
                </ProtectedRoute>
              }
            />

            {/* Catch-all */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
        <Toaster />
      </BrowserRouter>
    </QueryClientProvider>
  );
}
