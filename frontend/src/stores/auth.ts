import { create } from 'zustand';
import type { UserDTO } from '../types';

interface AuthState {
  user: UserDTO | null;
  isAuthenticated: boolean;
  login: (user: UserDTO, accessToken: string, refreshToken: string) => void;
  updateUser: (user: UserDTO) => void;
  logout: () => void;
  hydrate: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,

  login: (user, accessToken, refreshToken) => {
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
    localStorage.setItem('user', JSON.stringify(user));
    set({ user, isAuthenticated: true });
  },

  updateUser: (user) => {
    localStorage.setItem('user', JSON.stringify(user));
    set({ user });
  },

  logout: () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');
    set({ user: null, isAuthenticated: false });
  },

  hydrate: () => {
    const stored = localStorage.getItem('user');
    const token = localStorage.getItem('access_token');
    if (stored && token) {
      try {
        set({ user: JSON.parse(stored), isAuthenticated: true });
      } catch {
        localStorage.clear();
      }
    }
  },
}));
