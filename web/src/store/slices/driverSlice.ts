import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Driver, Trip } from '../../types';
import { TripEvents } from '../../contracts';

interface DriverState {
  driver: Driver | null;
  requestedTrip: Trip | null;
  tripStatus: TripEvents | null;
  error: string | null;
}

const initialState: DriverState = {
  driver: null,
  requestedTrip: null,
  tripStatus: null,
  error: null,
};

const driverSlice = createSlice({
  name: 'driver',
  initialState,
  reducers: {
    setDriver(state, action: PayloadAction<Driver | null>) {
      state.driver = action.payload;
    },
    setRequestedTrip(state, action: PayloadAction<Trip | null>) {
      state.requestedTrip = action.payload;
    },
    setTripStatus(state, action: PayloadAction<TripEvents | null>) {
      state.tripStatus = action.payload;
    },
    setError(state, action: PayloadAction<string | null>) {
      state.error = action.payload;
    },
    resetTrip(state) {
      state.tripStatus = null;
      state.requestedTrip = null;
    },
  },
});

export const {
  setDriver,
  setRequestedTrip,
  setTripStatus,
  setError,
  resetTrip,
} = driverSlice.actions;

export default driverSlice.reducer;
