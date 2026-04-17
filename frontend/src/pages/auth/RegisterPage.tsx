import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '../../stores/auth';
import { authAPI } from '../../api/services';
import { getErrorCode, getErrorMessage } from '../../api/client';
import { Button, Input } from '../../components/ui/Primitives';
import { useToastStore } from '../../stores/toast';

export default function RegisterPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [errors, setErrors] = useState<{ email?: string; password?: string; confirm?: string }>({});
  const [loading, setLoading] = useState(false);
  const { login } = useAuthStore();
  const navigate = useNavigate();
  const toast = useToastStore();

  const validate = () => {
    const e: typeof errors = {};
    if (!email) e.email = 'Email is required';
    else if (!/^[^\s@]+@nu\.edu\.kz$/i.test(email)) e.email = 'Must be a valid @nu.edu.kz email';
    if (!password) e.password = 'Password is required';
    else if (password.length < 8) e.password = 'Minimum 8 characters';
    if (password !== confirm) e.confirm = 'Passwords do not match';
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    setLoading(true);
    try {
      const { data } = await authAPI.register({ email, password });
      login(data.user, data.access_token, data.refresh_token);
      toast.add('Account created! Welcome to NU Events.', 'success');
      navigate('/events', { replace: true });
    } catch (err) {
      const code = getErrorCode(err);
      if (code === 'email_exists') {
        setErrors({ email: 'This email is already registered' });
      } else if (code === 'email_not_allowed') {
        setErrors({ email: 'Only @nu.edu.kz emails are allowed' });
      } else {
        toast.add(getErrorMessage(err), 'error');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[70vh] flex items-center justify-center">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-nu-gold flex items-center justify-center font-display font-black text-nu-dark text-2xl mx-auto mb-4">
            NU
          </div>
          <h1 className="font-display text-2xl font-bold">Create Account</h1>
          <p className="text-nu-text-muted text-sm mt-1">Register with your NU email</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="yourname@nu.edu.kz"
            error={errors.email}
          />
          <Input
            label="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Minimum 8 characters"
            error={errors.password}
          />
          <Input
            label="Confirm Password"
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            placeholder="Repeat password"
            error={errors.confirm}
          />
          <Button type="submit" loading={loading} className="w-full" size="lg">
            Create Account
          </Button>
        </form>

        <p className="text-center text-sm text-nu-text-muted mt-6">
          Already have an account?{' '}
          <Link to="/login" className="text-nu-gold hover:text-nu-gold-light font-medium">
            Sign In
          </Link>
        </p>
      </div>
    </div>
  );
}
