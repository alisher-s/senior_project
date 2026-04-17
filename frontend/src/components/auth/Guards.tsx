import { Navigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../../stores/auth';
import type { ReactNode } from 'react';

interface ProtectedRouteProps {
  children: ReactNode;
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { isAuthenticated } = useAuthStore();
  const location = useLocation();

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />;
  }
  return <>{children}</>;
}

interface RoleGuardProps {
  children: ReactNode;
  allow: string[];
}

export function RoleGuard({ children, allow }: RoleGuardProps) {
  const { user } = useAuthStore();

  if (!user || !allow.includes(user.role)) {
    return <Navigate to="/events" replace />;
  }
  return <>{children}</>;
}
