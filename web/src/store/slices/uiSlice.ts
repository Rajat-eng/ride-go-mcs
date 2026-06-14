import { createSlice, PayloadAction } from '@reduxjs/toolkit';

const ERROR_DEDUPE_WINDOW_MS = 8000;
const MAX_VISIBLE_ERRORS = 3;

export interface AppError {
  id: string;
  message: string;
  status?: number;
  timestamp: number;
}

interface UIState {
  errors: AppError[];
}

const initialState: UIState = {
  errors: [],
};

const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    addError(state, action: PayloadAction<Omit<AppError, 'id'>>) {
      const message = action.payload.message.trim();
      const timestamp = action.payload.timestamp;

      const recentlySeenSameError = state.errors.some(
        (error) => error.message === message && (timestamp - error.timestamp) < ERROR_DEDUPE_WINDOW_MS,
      );

      if (recentlySeenSameError) {
        return;
      }

      state.errors.push({
        ...action.payload,
        message,
        id: `${timestamp}-${Math.random().toString(36).slice(2, 9)}`,
      });

      if (state.errors.length > MAX_VISIBLE_ERRORS) {
        state.errors = state.errors.slice(state.errors.length - MAX_VISIBLE_ERRORS);
      }
    },
    dismissError(state, action: PayloadAction<string>) {
      state.errors = state.errors.filter((e) => e.id !== action.payload);
    },
    clearErrors(state) {
      state.errors = [];
    },
  },
});

export const { addError, dismissError, clearErrors } = uiSlice.actions;
export default uiSlice.reducer;
