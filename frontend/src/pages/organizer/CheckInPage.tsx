import { useState } from 'react';
import { ticketsAPI } from '../../api/services';
import { getErrorCode, getErrorMessage } from '../../api/client';
import { Button, Input } from '../../components/ui/Primitives';
import { ScanLine, CheckCircle, XCircle } from 'lucide-react';

type CheckInResult = {
  success: boolean;
  ticketId: string;
  message: string;
} | null;

export default function CheckInPage() {
  const [hash, setHash] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<CheckInResult>(null);

  const handleCheckIn = async () => {
    if (!hash.trim()) return;
    setLoading(true);
    setResult(null);

    try {
      const { data } = await ticketsAPI.use(hash.trim());
      setResult({
        success: true,
        ticketId: data.ticket_id,
        message: `Ticket verified! Status: ${data.status}`,
      });
      setHash('');
    } catch (err) {
      const code = getErrorCode(err);
      const messages: Record<string, string> = {
        ticket_not_found: 'Ticket not found',
        ticket_already_used: 'This ticket has already been used',
        ticket_already_cancelled: 'This ticket was cancelled',
        ticket_cannot_be_used: 'This ticket cannot be used',
        check_in_not_open: 'Check-in is not open yet',
      };
      setResult({
        success: false,
        ticketId: '',
        message: messages[code] || getErrorMessage(err),
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-lg mx-auto space-y-8">
      <div className="text-center">
        <ScanLine className="w-12 h-12 text-nu-gold mx-auto mb-4" />
        <h1 className="font-display text-2xl font-bold">Check-In</h1>
        <p className="text-nu-text-muted text-sm mt-1">
          Enter the QR hash from a student's ticket to verify entry
        </p>
      </div>

      <div className="rounded-2xl border border-white/5 bg-nu-surface/50 p-6">
        <div className="space-y-4">
          <Input
            label="QR Hash (hex)"
            value={hash}
            onChange={(e) => setHash(e.target.value)}
            placeholder="Paste QR hash hex value..."
            onKeyDown={(e) => e.key === 'Enter' && handleCheckIn()}
          />
          <Button onClick={handleCheckIn} loading={loading} className="w-full" size="lg">
            Verify Ticket
          </Button>
        </div>
      </div>

      {/* Result */}
      {result && (
        <div
          className={`rounded-2xl border p-6 text-center ${
            result.success
              ? 'border-nu-success/30 bg-nu-success/5'
              : 'border-nu-error/30 bg-nu-error/5'
          }`}
        >
          {result.success ? (
            <CheckCircle className="w-16 h-16 text-nu-success mx-auto mb-3" />
          ) : (
            <XCircle className="w-16 h-16 text-nu-error mx-auto mb-3" />
          )}
          <p className="font-display font-bold text-lg">{result.success ? 'Valid!' : 'Invalid'}</p>
          <p className="text-sm text-nu-text-muted mt-1">{result.message}</p>
        </div>
      )}
    </div>
  );
}
