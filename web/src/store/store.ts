import { configureStore, combineReducers } from '@reduxjs/toolkit';
import { useDispatch, useSelector, TypedUseSelectorHook } from 'react-redux';
import {
  persistStore,
  persistReducer,
  FLUSH,
  REHYDRATE,
  PAUSE,
  PERSIST,
  PURGE,
  REGISTER,
} from 'redux-persist';
import storage from 'redux-persist/lib/storage';
import riderReducer from './slices/riderSlice';
import driverReducer from './slices/driverSlice';
import uiReducer from './slices/uiSlice';
import authReducer from './slices/authSlice';
import { tripApi } from './api/tripApi';
import { authApi } from './api/authApi';
import { rtkQueryErrorMiddleware } from './middleware/errorMiddleware';

// Persist auth (session survival across refreshes) and active trip state for
// rider and driver so users land back on their in-progress trip after a reload.
const authPersistConfig = {
  key: 'auth',
  storage,
  whitelist: ['user', 'accessToken', 'isAuthenticated'],
};

const riderPersistConfig = {
  key: 'rider',
  storage,
  // omit ephemeral/high-frequency fields that should reset on reload
  blacklist: ['assignedDriverLocation', 'error'],
};

const driverPersistConfig = {
  key: 'driver',
  storage,
  blacklist: ['error'],
};

const rootReducer = combineReducers({
  rider: persistReducer(riderPersistConfig, riderReducer),
  driver: persistReducer(driverPersistConfig, driverReducer),
  ui: uiReducer,
  auth: persistReducer(authPersistConfig, authReducer),
  [tripApi.reducerPath]: tripApi.reducer,
  [authApi.reducerPath]: authApi.reducer,
});

export const store = configureStore({
  reducer: rootReducer,
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: {
        // redux-persist dispatches non-serialisable actions internally
        ignoredActions: [FLUSH, REHYDRATE, PAUSE, PERSIST, PURGE, REGISTER],
      },
    }).concat(tripApi.middleware, authApi.middleware, rtkQueryErrorMiddleware),
});

export const persistor = persistStore(store);

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

export const useAppDispatch: () => AppDispatch = useDispatch;
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector;
