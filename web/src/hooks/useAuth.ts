import { useCallback, useEffect, useRef, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../store/store';
import { setCredentials, logout } from '../store/slices/authSlice';
import apiClient from '../lib/axios';
import { useSignupMutation, useLoginMutation, SignupPayload, LoginPayload } from '../store/api/authApi';

export function useAuth() {
  const dispatch = useAppDispatch();
  const { user, isAuthenticated, accessToken } = useAppSelector((s) => s.auth);
  const [isRestoringSession, setIsRestoringSession] = useState(true);
  const restoreAttemptedRef = useRef(false);
  // Capture token at mount time via ref so the effect doesn't need it as a dependency
  const initialTokenRef = useRef(accessToken);

  const [signupMutation, { isLoading: isSigningUp }] = useSignupMutation();
  const [loginMutation, { isLoading: isLoggingIn }] = useLoginMutation();

  useEffect(() => {
    if (restoreAttemptedRef.current) {
      return;
    }
    restoreAttemptedRef.current = true;

    // Already authenticated (e.g., just set by OAuth callback) — no need to hit refresh
    if (initialTokenRef.current) {
      setIsRestoringSession(false);
      return;
    }

    const restore = async () => {
      try {
        const result = await apiClient.post('/auth/refresh');
        const payload = result.data?.data;

        if (payload?.accessToken && payload?.user) {
          dispatch(setCredentials({
            user: payload.user,
            accessToken: payload.accessToken,
          }));
        }
      } catch {
        // No valid refresh cookie/session; user remains logged out.
      } finally {
        setIsRestoringSession(false);
      }
    };

    restore();
  }, [dispatch]);

  const signup = useCallback(async (payload: SignupPayload) => {
    const result = await signupMutation(payload).unwrap();
    dispatch(setCredentials({
      user: result.data.user,
      accessToken: result.data.accessToken,
    }));
    return result;
  }, [signupMutation, dispatch]);

  const login = useCallback(async (payload: LoginPayload) => {
    const result = await loginMutation(payload).unwrap();
    dispatch(setCredentials({
      user: result.data.user,
      accessToken: result.data.accessToken,
    }));
    return result;
  }, [loginMutation, dispatch]);

  const handleLogout = useCallback(async () => {
    // Ask the backend to clear the HttpOnly refresh_token cookie
    await apiClient.post('/auth/logout').catch(() => {});
    dispatch(logout());
  }, [dispatch]);

  return {
    user,
    isAuthenticated,
    accessToken,
    signup,
    login,
    logout: handleLogout,
    isSigningUp,
    isLoggingIn,
    isRestoringSession,
  };
}
