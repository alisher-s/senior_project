import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { eventsAPI } from '../../api/services';
import { getErrorMessage } from '../../api/client';
import { useToastStore } from '../../stores/toast';
import { Button, Input, Textarea, Spinner } from '../../components/ui/Primitives';
import { ArrowLeft } from 'lucide-react';

export default function EditEventPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToastStore();
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({
    title: '',
    description: '',
    cover_image_url: '',
    starts_at: '',
    capacity_total: '',
    status: '' as string,
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const { data: event, isLoading } = useQuery({
    queryKey: ['event', id],
    queryFn: () => eventsAPI.getById(id!).then((r) => r.data),
    enabled: !!id,
  });

  useEffect(() => {
    if (event) {
      const dt = new Date(event.starts_at);
      const local = new Date(dt.getTime() - dt.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
      setForm({
        title: event.title,
        description: event.description,
        cover_image_url: event.cover_image_url || '',
        starts_at: local,
        capacity_total: String(event.capacity_total),
        status: event.status,
      });
    }
  }, [event]);

  const update = (field: string, value: string) => {
    setForm((f) => ({ ...f, [field]: value }));
    setErrors((e) => ({ ...e, [field]: '' }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id) return;

    setLoading(true);
    try {
      await eventsAPI.update(id, {
        title: form.title,
        description: form.description,
        cover_image_url: form.cover_image_url || undefined,
        starts_at: new Date(form.starts_at).toISOString(),
        capacity_total: parseInt(form.capacity_total),
        status: form.status as 'draft' | 'published' | 'cancelled',
      });
      toast.add('Event updated.', 'success');
      navigate('/organizer');
    } catch (err) {
      toast.add(getErrorMessage(err), 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!id || !confirm('Are you sure you want to delete this event?')) return;
    try {
      await eventsAPI.delete(id);
      toast.add('Event deleted.', 'info');
      navigate('/organizer');
    } catch (err) {
      toast.add(getErrorMessage(err), 'error');
    }
  };

  if (isLoading) {
    return <div className="flex justify-center py-20"><Spinner className="w-8 h-8" /></div>;
  }

  return (
    <div className="max-w-2xl mx-auto">
      <button
        onClick={() => navigate('/organizer')}
        className="flex items-center gap-2 text-sm text-nu-text-muted hover:text-nu-text mb-6 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" /> Back to Dashboard
      </button>

      <div className="rounded-2xl border border-white/5 bg-nu-surface/50 p-6 sm:p-8">
        <h1 className="font-display text-2xl font-bold mb-6">Edit Event</h1>

        <form onSubmit={handleSubmit} className="space-y-5">
          <Input
            label="Event Title"
            value={form.title}
            onChange={(e) => update('title', e.target.value)}
            error={errors.title}
          />

          <Textarea
            label="Description"
            value={form.description}
            onChange={(e) => update('description', e.target.value)}
            rows={5}
            error={errors.description}
          />

          <Input
            label="Cover Image URL (optional)"
            value={form.cover_image_url}
            onChange={(e) => update('cover_image_url', e.target.value)}
            placeholder="https://example.com/image.jpg"
          />

          <Input
            label="Start Date & Time"
            type="datetime-local"
            value={form.starts_at}
            onChange={(e) => update('starts_at', e.target.value)}
            error={errors.starts_at}
          />

          <Input
            label="Total Capacity"
            type="number"
            value={form.capacity_total}
            onChange={(e) => update('capacity_total', e.target.value)}
            min={1}
            max={100000}
            error={errors.capacity_total}
          />

          {/* Status selector */}
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium text-nu-text-muted">Status</label>
            <select
              value={form.status}
              onChange={(e) => update('status', e.target.value)}
              className="w-full rounded-xl border border-white/10 bg-nu-surface px-4 py-2.5 text-sm text-nu-text focus:outline-none focus:ring-2 focus:ring-nu-gold/50 transition-all"
            >
              <option value="draft">Draft</option>
              <option value="published">Published</option>
              <option value="cancelled">Cancelled</option>
            </select>
          </div>

          <div className="flex gap-3 pt-4">
            <Button type="submit" loading={loading} size="lg">
              Save Changes
            </Button>
            <Button type="button" variant="danger" onClick={handleDelete}>
              Delete Event
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
