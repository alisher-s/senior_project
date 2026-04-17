import { useToastStore } from '../../stores/toast';
import { X, CheckCircle, AlertCircle, Info } from 'lucide-react';

const icons = {
  success: CheckCircle,
  error: AlertCircle,
  info: Info,
};

const colors = {
  success: 'border-nu-success/40 bg-nu-success/10',
  error: 'border-nu-error/40 bg-nu-error/10',
  info: 'border-nu-gold/40 bg-nu-gold/10',
};

export default function Toaster() {
  const { toasts, remove } = useToastStore();

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col gap-3 max-w-sm">
      {toasts.map((t) => {
        const Icon = icons[t.type];
        return (
          <div
            key={t.id}
            className={`flex items-start gap-3 rounded-xl border px-4 py-3 shadow-2xl backdrop-blur-md animate-[slideIn_0.3s_ease-out] ${colors[t.type]}`}
          >
            <Icon className="w-5 h-5 mt-0.5 shrink-0" />
            <p className="text-sm flex-1">{t.message}</p>
            <button onClick={() => remove(t.id)} className="text-nu-text-muted hover:text-nu-text">
              <X className="w-4 h-4" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
