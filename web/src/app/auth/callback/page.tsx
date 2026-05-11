'use client';

import { Suspense, useEffect, useRef, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useSession } from 'next-auth/react';
import { API_URL } from '../../../constants';
import { useAppDispatch } from '../../../store/store';
import { setCredentials } from '../../../store/slices/authSlice';

interface OAuthSessionResponse {
  data: {
    accessToken: string;
    user: {
      id: string;
      email: string;
      name: string;
      phoneNumber: string;
      role: 'rider' | 'driver';
    };
  };
}

function OAuthCallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { data: session, status } = useSession();
  const dispatch = useAppDispatch();
  const [error, setError] = useState('');
  const exchangedRef = useRef(false);

  const selectedRole = searchParams.get('role') === 'driver' ? 'driver' : 'rider';

  useEffect(() => {
    if (status === 'loading' || exchangedRef.current) {
      return;
    }

    if (status === 'unauthenticated') {
      setError('Google authentication failed. Please try again.');
      return;
    }

    const googleIDToken = session?.googleIdToken;
    if (!googleIDToken) {
      setError('Google token missing in session. Please sign in again.');
      return;
    }

    exchangedRef.current = true;

    const consumeSession = async () => {
      try {
        const response = await fetch(`${API_URL}/auth/google`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include',
          body: JSON.stringify({ idToken: googleIDToken, role: selectedRole }),
        });

        if (!response.ok) {
          throw new Error('Backend Google auth exchange failed');
        }

        const payload = (await response.json()) as OAuthSessionResponse;
        dispatch(setCredentials({
          user: payload.data.user,
          accessToken: payload.data.accessToken,
        }));

        router.replace('/');
      } catch {
        exchangedRef.current = false;
        setError('Unable to complete Google login. Please try again.');
      }
    };

    consumeSession();
  }, [dispatch, router, selectedRole, session, status]);

  if (error) {
    return (
      <main className="min-h-screen bg-gradient-to-b from-white to-gray-50">
        <div className="flex flex-col items-center justify-center h-screen gap-4 px-4">
          <div className="bg-white p-8 rounded-2xl shadow-lg text-center max-w-md w-full">
            <h1 className="text-xl font-semibold text-gray-900">Google Login Failed</h1>
            <p className="text-gray-600 mt-3">{error}</p>
            <button
              type="button"
              onClick={() => router.replace('/')}
              className="mt-6 px-4 py-2 rounded-md border border-gray-300 hover:bg-gray-50"
            >
              Back to Home
            </button>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-gradient-to-b from-white to-gray-50">
      <div className="flex flex-col items-center justify-center h-screen gap-4">
        <div className="bg-white p-8 rounded-2xl shadow-lg text-center max-w-md w-full">
          <h1 className="text-xl font-semibold text-gray-900">Completing Google Login</h1>
          <p className="text-gray-600 mt-3">Please wait while we finalize your session.</p>
        </div>
      </div>
    </main>
  );
}

export default function OAuthCallbackPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen bg-gradient-to-b from-white to-gray-50">
          <div className="flex flex-col items-center justify-center h-screen gap-4">
            <div className="bg-white p-8 rounded-2xl shadow-lg text-center max-w-md w-full">
              <h1 className="text-xl font-semibold text-gray-900">Completing Google Login</h1>
              <p className="text-gray-600 mt-3">Please wait while we finalize your session.</p>
            </div>
          </div>
        </main>
      }
    >
      <OAuthCallbackContent />
    </Suspense>
  );
}
