import { useCallback, useEffect, useRef, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../store/store';
import { setCredentials, logout } from '../store/slices/authSlice';
import { clearState as clearDriverState } from '../store/slices/driverSlice';
import { clearState as clearRiderState } from '../store/slices/riderSlice';
import apiClient from '../lib/axios';
import { useSignupMutation, useLoginMutation, SignupPayload, LoginPayload } from '../store/api/authApi';
import { RefreshResponseSchema } from '../lib/schemas';

export function useAuth() {
  const dispatch = useAppDispatch();
  const { user, isAuthenticated, accessToken } = useAppSelector((s) => s.auth);
  const riderState = useAppSelector((s) => s.rider);
  const driverState = useAppSelector((s) => s.driver);
  const [isRestoringSession, setIsRestoringSession] = useState(true);
  const restoreAttemptedRef = useRef(false); 

  const [signupMutation, { isLoading: isSigningUp }] = useSignupMutation();
  const [loginMutation, { isLoading: isLoggingIn }] = useLoginMutation();

  useEffect(() => {
    if (restoreAttemptedRef.current) {
      return;
    }
    restoreAttemptedRef.current = true;

    const restore = async () => {
      try {
        const result = await apiClient.post('/auth/refresh');
        const parsed = RefreshResponseSchema.safeParse(result.data);

        if (parsed.success) {
          dispatch(setCredentials({
            user: parsed.data.data.user,
            accessToken: parsed.data.data.accessToken,
          }));
        } else {
          dispatch(logout());
        }
      } catch {
        // No valid refresh cookie/session; clear any persisted client auth state.
        dispatch(logout());
      } finally {
        setIsRestoringSession(false);
      }
    };

    restore();
  }, [dispatch]);

  useEffect(() => {
    if (isRestoringSession) {
      return;
    }

    const hasRiderState = Boolean(
      riderState.trip ||
      riderState.destination ||
      riderState.paymentSession ||
      riderState.assignedDriver ||
      riderState.chatMessages.length ||
      riderState.drivers.length,
    );

    const hasDriverState = Boolean(
      driverState.driver ||
      driverState.requestedTrip ||
      driverState.chatMessages.length,
    );

    if (!isAuthenticated || !user) {
      if (hasRiderState || riderState.ownerUserID) {
        dispatch(clearRiderState());
      }
      if (hasDriverState || driverState.ownerUserID) {
        dispatch(clearDriverState());
      }
      return;
    }

    if (user.role === 'rider') {
      if (hasDriverState || driverState.ownerUserID) {
        dispatch(clearDriverState());
      }

      if (
        riderState.ownerUserID !== user.id &&
        (riderState.ownerUserID !== null || hasRiderState)
      ) {
        dispatch(clearRiderState());
      }

      return;
    }

    if (hasRiderState || riderState.ownerUserID) {
      dispatch(clearRiderState());
    }

    if (
      driverState.ownerUserID !== user.id &&
      (driverState.ownerUserID !== null || hasDriverState)
    ) {
      dispatch(clearDriverState());
    }
  }, [dispatch, driverState, isAuthenticated, isRestoringSession, riderState, user]);

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
