import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { eventsAPI } from '../../api/services';
import { getErrorMessage } from '../../api/client';
import { useToastStore } from '../../stores/toast';
import { Button, Input, Textarea } from '../../components/ui/Primitives';
import { ArrowLeft } from 'lucide-react';

export default function CreateEventPage() {
  const navigate = useNavigate();
  const toast = useToastStore();
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({
    title: '',
    description: '',
    cover_image_url: '',
    starts_at: '',
    capacity_total: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const update = (field: string, value: string) => {
    setForm((f) => ({ ...f, [field]: value }));
    setErrors((e) => ({ ...e, [field]: '' }));
  };

  const validate = () => {
    const e: Record<string, string> = {};
    if (!form.title || form.title.length < 3) e.title = 'Title must be at least 3 characters';
    if (form.title.length > 120) e.title = 'Title must be under 120 characters';
    if (!form.starts_at) e.starts_at = 'Start date is required';
    else if (new Date(form.starts_at) <= new Date()) e.starts_at = 'Must be in the future';
    if (!form.capacity_total) e.capacity_total = 'Capacity is required';
    else {
      const cap = parseInt(form.capacity_total);
      if (isNaN(cap) || cap < 1) e.capacity_total = 'Must be at least 1';
      if (cap > 100000) e.capacity_total = 'Maximum 100,000';
    }
    if (form.description.length > 2000) e.description = 'Max 2000 characters';
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    setLoading(true);
    try {
      const payload = {
        title: form.title,
        description: form.description,
        cover_image_url: form.cover_image_url || undefined,
        starts_at: new Date(form.starts_at).toISOString(),
        capacity_total: parseInt(form.capacity_total),
      };
      await eventsAPI.create(payload);
      toast.add('Event created! It will be visible after admin approval.', 'success');
      navigate('/organizer');
    } catch (err) {
      toast.add(getErrorMessage(err), 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto">
      <button
        onClick={() => navigate('/organizer')}
        className="flex items-center gap-2 text-sm text-nu-text-muted hover:text-nu-text mb-6 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" /> Back to Dashboard
      </button>

      <div className="rounded-2xl border border-white/5 bg-nu-surface/50 p-6 sm:p-8">
        <h1 className="font-display text-2xl font-bold mb-1">Create Event</h1>
        <p className="text-nu-text-muted text-sm mb-6">
          New events require admin approval before becoming visible.
        </p>

        <form onSubmit={handleSubmit} className="space-y-5">
          <Input
            label="Event Title"
            value={form.title}
            onChange={(e) => update('title', e.target.value)}
            placeholder="e.g. NU Hackathon 2026"
            error={errors.title}
          />

          <Textarea
            label="Description"
            value={form.description}
            onChange={(e) => update('description', e.target.value)}
            placeholder="Describe your event..."
            rows={5}
            error={errors.description}
          />

          <Input
            label="Cover Image URL (optional)"
            value={form.cover_image_url}
            onChange={(e) => update('cover_image_url', e.target.value)}
            placeholder="https://example.com/image.jpg"
            error={errors.cover_image_url}
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
            placeholder="e.g. 100"
            min={1}
            max={100000}
            error={errors.capacity_total}
          />

          <div className="flex gap-3 pt-4">
            <Button type="submit" loading={loading} size="lg">
              Create Event
            </Button>
            <Button type="button" variant="ghost" onClick={() => navigate('/organizer')}>
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
