import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../stores/auth';
import {
  CalendarDays, Ticket, LayoutDashboard, Shield, LogOut, Menu, X, Plus,
} from 'lucide-react';
import { useState } from 'react';
import { cn } from '../../lib/utils';

export default function AppShell() {
  const { user, isAuthenticated, logout } = useAuthStore();
  const location = useLocation();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const navItems = [
    { to: '/events', label: 'Events', icon: CalendarDays, roles: null },
    { to: '/my/tickets', label: 'My Tickets', icon: Ticket, roles: ['student', 'organizer', 'admin'] },
    { to: '/organizer', label: 'Dashboard', icon: LayoutDashboard, roles: ['organizer', 'admin'] },
    { to: '/organizer/events/new', label: 'Create Event', icon: Plus, roles: ['organizer', 'admin'] },
    { to: '/admin', label: 'Admin', icon: Shield, roles: ['admin'] },
  ].filter(
    (item) => item.roles === null || (user && item.roles.includes(user.role))
  );

  const isActive = (path: string) => location.pathname === path || location.pathname.startsWith(path + '/');

  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="sticky top-0 z-40 border-b border-white/5 bg-nu-dark/80 backdrop-blur-xl">
        <div className="max-w-7xl mx-auto flex items-center justify-between px-4 sm:px-6 h-16">
          <Link to="/" className="flex items-center gap-3 group">
            <div className="w-8 h-8 rounded-lg bg-nu-gold flex items-center justify-center font-display font-black text-nu-dark text-sm">
              NU
            </div>
            <span className="font-display font-bold text-lg hidden sm:block">
              Events
            </span>
          </Link>

          {/* Desktop nav */}
          <nav className="hidden md:flex items-center gap-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.to}
                  to={item.to}
                  className={cn(
                    'flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-all',
                    isActive(item.to)
                      ? 'bg-nu-gold/15 text-nu-gold font-semibold'
                      : 'text-nu-text-muted hover:text-nu-text hover:bg-white/5'
                  )}
                >
                  <Icon className="w-4 h-4" />
                  {item.label}
                </Link>
              );
            })}
          </nav>

          <div className="flex items-center gap-3">
            {isAuthenticated ? (
              <div className="flex items-center gap-3">
                <div className="hidden sm:flex flex-col items-end">
                  <span className="text-sm font-medium">{user?.email.split('@')[0]}</span>
                  <span className="text-xs text-nu-gold capitalize">{user?.role}</span>
                </div>
                <button
                  onClick={handleLogout}
                  className="p-2 rounded-lg text-nu-text-muted hover:text-nu-text hover:bg-white/5 transition-all"
                  title="Logout"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <Link
                to="/login"
                className="text-sm font-semibold px-4 py-2 rounded-xl bg-nu-gold text-nu-dark hover:bg-nu-gold-light transition-all"
              >
                Sign In
              </Link>
            )}

            {/* Mobile hamburger */}
            <button
              onClick={() => setMenuOpen(!menuOpen)}
              className="md:hidden p-2 rounded-lg text-nu-text-muted hover:text-nu-text"
            >
              {menuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
            </button>
          </div>
        </div>

        {/* Mobile nav */}
        {menuOpen && (
          <div className="md:hidden border-t border-white/5 bg-nu-dark/95 backdrop-blur-xl">
            <nav className="flex flex-col p-4 gap-1">
              {navItems.map((item) => {
                const Icon = item.icon;
                return (
                  <Link
                    key={item.to}
                    to={item.to}
                    onClick={() => setMenuOpen(false)}
                    className={cn(
                      'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm',
                      isActive(item.to)
                        ? 'bg-nu-gold/15 text-nu-gold font-semibold'
                        : 'text-nu-text-muted'
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    {item.label}
                  </Link>
                );
              })}
            </nav>
          </div>
        )}
      </header>

      {/* Main content */}
      <main className="flex-1">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
          <Outlet />
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-white/5 py-6 text-center text-xs text-nu-text-muted">
        Nazarbayev University · Student Event Ticketing Platform · 2026
      </footer>
    </div>
  );
}
