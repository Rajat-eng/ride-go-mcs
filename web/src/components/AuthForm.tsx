'use client';

import { useState } from 'react';
import { Button } from './ui/button';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { useAuth } from '../hooks/useAuth';

interface AuthFormProps {
  role: 'driver' | 'rider';
  onSuccess: () => void;
  onBack: () => void;
}

export function AuthForm({ role, onSuccess, onBack }: AuthFormProps) {
  const [mode, setMode] = useState<'login' | 'signup'>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');
  const [phoneNumber, setPhoneNumber] = useState('');
  const [error, setError] = useState('');

  const { signup, login, isSigningUp, isLoggingIn } = useAuth();

  const isLoading = isSigningUp || isLoggingIn;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    try {
      if (mode === 'signup') {
        await signup({ email, password, name, phoneNumber });
      } else {
        await login({ email, password });
      }
      onSuccess();
    } catch {
      setError(mode === 'signup' ? 'Failed to create account. Email may already be in use.' : 'Invalid email or password.');
    }
  };

  return (
    <div className="flex flex-col items-center justify-center h-screen gap-6 px-4">
      <Card className="max-w-md w-full">
        <CardHeader>
          <CardTitle className="text-center">
            {mode === 'login' ? 'Login' : 'Sign Up'} as {role === 'rider' ? 'Rider' : 'Driver'}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {mode === 'signup' && (
              <>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Phone Number</label>
                  <input
                    type="tel"
                    value={phoneNumber}
                    onChange={(e) => setPhoneNumber(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </>
            )}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
                required
                minLength={8}
              />
            </div>

            {error && (
              <p className="text-red-500 text-sm">{error}</p>
            )}

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? 'Please wait...' : mode === 'login' ? 'Login' : 'Create Account'}
            </Button>
          </form>

          <div className="mt-4 text-center">
            <button
              type="button"
              onClick={() => { setMode(mode === 'login' ? 'signup' : 'login'); setError(''); }}
              className="text-sm text-primary hover:underline"
            >
              {mode === 'login' ? "Don't have an account? Sign up" : 'Already have an account? Login'}
            </button>
          </div>

          <div className="mt-4">
            <Button variant="outline" className="w-full" onClick={onBack}>
              Back
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
