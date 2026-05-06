import { isRejectedWithValue, Middleware } from '@reduxjs/toolkit';
import { addError } from '../slices/uiSlice';

export const rtkQueryErrorMiddleware: Middleware = (api) => (next) => (action) => {
  if (isRejectedWithValue(action)) {
    const payload = action.payload as { status?: number; data?: unknown };
    const message =
      (payload.data && typeof payload.data === 'object' && 'message' in payload.data
        ? (payload.data as { message: string }).message
        : null) ||
      (typeof payload.data === 'string' ? payload.data : null) ||
      'An API request failed';

    api.dispatch(
      addError({
        message,
        status: payload.status,
        timestamp: Date.now(),
      }),
    );
  }

  return next(action);
};
