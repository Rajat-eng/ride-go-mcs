import { configureStore } from '@reduxjs/toolkit';
import { useDispatch, useSelector, TypedUseSelectorHook } from 'react-redux';
import riderReducer from './slices/riderSlice';
import driverReducer from './slices/driverSlice';
import uiReducer from './slices/uiSlice';
import authReducer from './slices/authSlice';
import { tripApi } from './api/tripApi';
import { authApi } from './api/authApi';
import { rtkQueryErrorMiddleware } from './middleware/errorMiddleware';

export const store = configureStore({
  reducer: {
    rider: riderReducer,
    driver: driverReducer,
    ui: uiReducer,
    auth: authReducer,
    [tripApi.reducerPath]: tripApi.reducer,
    [authApi.reducerPath]: authApi.reducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(tripApi.middleware, authApi.middleware, rtkQueryErrorMiddleware),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

export const useAppDispatch: () => AppDispatch = useDispatch;
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector;
