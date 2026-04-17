import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { eventsAPI, adminAPI } from '../../api/services';
import { getErrorMessage } from '../../api/client';
import { useToastStore } from '../../stores/toast';
import { Button, Badge, Spinner, EmptyState, Modal, Input, Textarea } from '../../components/ui/Primitives';
import { formatEventDate } from '../../lib/utils';
import { Shield, CheckCircle, XCircle, CalendarDays, Users, UserCog } from 'lucide-react';
import type { EventDTO } from '../../types';

export default function AdminPanel() {
  const [tab, setTab] = useState<'moderation' | 'users'>('moderation');

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-display text-2xl font-bold">Admin Panel</h1>
        <p className="text-nu-text-muted text-sm mt-1">Moderate events and manage users</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 rounded-xl bg-nu-surface/50 w-fit">
        <button
          onClick={() => setTab('moderation')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
            tab === 'moderation' ? 'bg-nu-gold text-nu-dark' : 'text-nu-text-muted hover:text-nu-text'
          }`}
        >
          <Shield className="w-4 h-4 inline mr-2" />
          Event Moderation
        </button>
        <button
          onClick={() => setTab('users')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
            tab === 'users' ? 'bg-nu-gold text-nu-dark' : 'text-nu-text-muted hover:text-nu-text'
          }`}
        >
          <UserCog className="w-4 h-4 inline mr-2" />
          User Roles
        </button>
      </div>

      {tab === 'moderation' ? <ModerationQueue /> : <UserRoleManager />}
    </div>
  );
}

function ModerationQueue() {
  const queryClient = useQueryClient();
  const toast = useToastStore();
  const [selected, setSelected] = useState<EventDTO | null>(null);
  const [action, setAction] = useState<'approve' | 'reject'>('approve');
  const [reason, setReason] = useState('');
  const [loading, setLoading] = useState(false);

  // Fetch all events - we'll filter pending ones client-side since the public list only returns approved
  // In a real app, admin would have a separate endpoint for pending events
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['events', 'all-admin'],
    queryFn: () => eventsAPI.list({ limit: 100 }).then((r) => r.data),
  });

  const events = data?.items || [];

  const handleModerate = async () => {
    if (!selected) return;
    setLoading(true);
    try {
      await adminAPI.moderateEvent(selected.id, {
        action,
        reason: reason || undefined,
      });
      toast.add(
        action === 'approve' ? 'Event approved!' : 'Event rejected.',
        action === 'approve' ? 'success' : 'info'
      );
      setSelected(null);
      setReason('');
      queryClient.invalidateQueries({ queryKey: ['events'] });
      refetch();
    } catch (err) {
      toast.add(getErrorMessage(err), 'error');
    } finally {
      setLoading(false);
    }
  };

  if (isLoading) {
    return <div className="flex justify-center py-12"><Spinner className="w-8 h-8" /></div>;
  }

  return (
    <>
      <div className="space-y-3">
        <h2 className="font-display font-semibold">All Events</h2>
        {events.length === 0 ? (
          <EmptyState
            icon={<Shield className="w-12 h-12" />}
            title="No events to moderate"
            description="All events have been reviewed."
          />
        ) : (
          events.map((event) => (
            <div
              key={event.id}
              className="flex items-center gap-4 rounded-xl border border-white/5 bg-nu-surface/50 p-4"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <h3 className="font-semibold text-sm truncate">{event.title}</h3>
                  <Badge
                    variant={
                      event.moderation_status === 'approved' ? 'success' :
                      event.moderation_status === 'rejected' ? 'error' : 'warning'
                    }
                  >
                    {event.moderation_status}
                  </Badge>
                </div>
                <div className="flex items-center gap-4 text-xs text-nu-text-muted">
                  <span className="flex items-center gap-1">
                    <CalendarDays className="w-3 h-3" />
                    {formatEventDate(event.starts_at)}
                  </span>
                  <span className="flex items-center gap-1">
                    <Users className="w-3 h-3" />
                    {event.capacity_total} seats
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-2 shrink-0">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => { setSelected(event); setAction('approve'); }}
                >
                  <CheckCircle className="w-3.5 h-3.5 text-nu-success" />
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => { setSelected(event); setAction('reject'); }}
                >
                  <XCircle className="w-3.5 h-3.5 text-nu-error" />
                </Button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Moderate modal */}
      <Modal
        open={!!selected}
        onClose={() => setSelected(null)}
        title={action === 'approve' ? 'Approve Event' : 'Reject Event'}
      >
        {selected && (
          <div className="space-y-4">
            <p className="text-sm text-nu-text-muted">
              {action === 'approve'
                ? `Approve "${selected.title}"? It will become visible to all students.`
                : `Reject "${selected.title}"? The organizer will be notified.`}
            </p>

            {action === 'reject' && (
              <Textarea
                label="Reason (optional)"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Why is this event being rejected?"
                rows={3}
              />
            )}

            <div className="flex gap-3 justify-end">
              <Button variant="ghost" onClick={() => setSelected(null)}>Cancel</Button>
              <Button
                variant={action === 'approve' ? 'primary' : 'danger'}
                onClick={handleModerate}
                loading={loading}
              >
                {action === 'approve' ? 'Approve' : 'Reject'}
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </>
  );
}

function UserRoleManager() {
  const toast = useToastStore();
  const [userId, setUserId] = useState('');
  const [role, setRole] = useState<'student' | 'organizer' | 'admin'>('student');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ id: string; email: string; role: string } | null>(null);

  const handleSetRole = async () => {
    if (!userId.trim()) return;
    setLoading(true);
    setResult(null);
    try {
      const { data } = await adminAPI.setUserRole(userId.trim(), { role });
      setResult(data);
      toast.add(`Role updated to ${data.role} for ${data.email}`, 'success');
      setUserId('');
    } catch (err) {
      toast.add(getErrorMessage(err), 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-lg space-y-6">
      <h2 className="font-display font-semibold">Change User Role</h2>

      <div className="rounded-2xl border border-white/5 bg-nu-surface/50 p-6 space-y-4">
        <Input
          label="User ID (UUID)"
          value={userId}
          onChange={(e) => setUserId(e.target.value)}
          placeholder="e.g. 550e8400-e29b-41d4-a716-446655440000"
        />

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-nu-text-muted">New Role</label>
          <select
            value={role}
            onChange={(e) => setRole(e.target.value as typeof role)}
            className="w-full rounded-xl border border-white/10 bg-nu-surface px-4 py-2.5 text-sm text-nu-text focus:outline-none focus:ring-2 focus:ring-nu-gold/50 transition-all"
          >
            <option value="student">Student</option>
            <option value="organizer">Organizer</option>
            <option value="admin">Admin</option>
          </select>
        </div>

        <Button onClick={handleSetRole} loading={loading} className="w-full">
          Update Role
        </Button>
      </div>

      {result && (
        <div className="rounded-xl border border-nu-success/20 bg-nu-success/5 p-4">
          <p className="text-sm">
            <strong>{result.email}</strong> is now a <Badge variant="gold">{result.role}</Badge>
          </p>
        </div>
      )}
    </div>
  );
}
