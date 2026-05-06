import { useCallback } from 'react';
import { useAppDispatch, useAppSelector } from '../store/store';
import { setCredentials, logout } from '../store/slices/authSlice';
import { useSignupMutation, useLoginMutation, SignupPayload, LoginPayload } from '../store/api/authApi';

export function useAuth() {
  const dispatch = useAppDispatch();
  const { user, isAuthenticated, accessToken } = useAppSelector((s) => s.auth);

  const [signupMutation, { isLoading: isSigningUp }] = useSignupMutation();
  const [loginMutation, { isLoading: isLoggingIn }] = useLoginMutation();

  const signup = useCallback(async (payload: SignupPayload) => {
    const result = await signupMutation(payload).unwrap();
    dispatch(setCredentials({
      user: result.data.user,
      accessToken: result.data.accessToken,
      refreshToken: result.data.refreshToken,
    }));
    return result;
  }, [signupMutation, dispatch]);

  const login = useCallback(async (payload: LoginPayload) => {
    const result = await loginMutation(payload).unwrap();
    dispatch(setCredentials({
      user: result.data.user,
      accessToken: result.data.accessToken,
      refreshToken: result.data.refreshToken,
    }));
    return result;
  }, [loginMutation, dispatch]);

  const handleLogout = useCallback(() => {
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
  };
}
