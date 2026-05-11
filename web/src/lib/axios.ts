import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { API_URL } from '../constants';
import { store } from '../store/store';
import { addError } from '../store/slices/uiSlice';
import { setCredentials, logout } from '../store/slices/authSlice';

const apiClient = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // send the HttpOnly refresh_token cookie on every request
});

// Attach access token to every request
apiClient.interceptors.request.use((config) => {
  const token = store.getState().auth.accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

let isRefreshing = false;
let failedQueue: { 
  resolve: (token: string) => void; 
  reject: (err: unknown) => void 
}[] = [];

function processQueue(error: unknown, token: string | null) {
  // Queue all the failed requests and retry them with the new token
  failedQueue.forEach(({ resolve, reject }) => {
    if (token) {
      resolve(token);
    } else {
      reject(error);
    }
  });
  failedQueue = []; // Clear the queue after processing
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    // If 401 and not already retrying, attempt token refresh
    if (error.response?.status === 401 && !originalRequest._retry) {
      // If the refresh endpoint itself returned 401, the refresh token is invalid – logout
      // BUT: skip logout if user already has valid credentials (e.g., set by OAuth callback before restore ran)
      if (originalRequest.url === '/auth/refresh') {
        const currentToken = store.getState().auth.accessToken;
        if (!currentToken) {
          store.dispatch(logout());
        }
        return Promise.reject(error);
      }

      if (isRefreshing) {
        // Queue the request until refresh completes
        return new Promise<string>((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        }).then((token) => {
          originalRequest.headers.Authorization = `Bearer ${token}`;
          return apiClient(originalRequest);
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        // No body needed — the browser sends the HttpOnly refresh_token cookie automatically
        const { data } = await axios.post(`${API_URL}/auth/refresh`, null, { withCredentials: true });
        const newAccessToken = data.data.accessToken;

        store.dispatch(setCredentials({
          user: store.getState().auth.user!,
          accessToken: newAccessToken,
        }));

        processQueue(null, newAccessToken);

        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
        return apiClient(originalRequest);
      } catch (refreshError) {
        processQueue(refreshError, null); // Reject all queued requests
        store.dispatch(logout());
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    // Non-401 errors: dispatch to error toast
    const message =
      (error.response?.data as { message?: string })?.message ||
      (error.response?.data as { error?: string })?.error ||
      error.message ||
      'An unexpected error occurred';

    const status = error.response?.status;

    store.dispatch(
      addError({
        message,
        status,
        timestamp: Date.now(),
      }),
    );

    return Promise.reject(error);
  },
);

export default apiClient;
