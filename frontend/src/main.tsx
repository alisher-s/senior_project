import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './index.css';
import { useAuthStore } from './stores/auth';
import { useTicketsStore } from './stores/tickets';

// Hydrate stores from localStorage before first render
useAuthStore.getState().hydrate();
useTicketsStore.getState().hydrate();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
