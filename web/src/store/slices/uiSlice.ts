import { createSlice, PayloadAction } from '@reduxjs/toolkit';

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
      state.errors.push({
        ...action.payload,
        id: `${action.payload.timestamp}-${Math.random().toString(36).slice(2, 9)}`,
      });
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
