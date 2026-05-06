import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Driver, TripPreview } from '../../types';
import { PaymentEventSessionCreatedData, TripEvents } from '../../contracts';

interface RiderState {
  drivers: Driver[];
  tripStatus: TripEvents | null;
  paymentSession: PaymentEventSessionCreatedData | null;
  assignedDriver: Driver | null;
  trip: TripPreview | null;
  destination: [number, number] | null;
  error: string | null;
}

const initialState: RiderState = {
  drivers: [],
  tripStatus: null,
  paymentSession: null,
  assignedDriver: null,
  trip: null,
  destination: null,
  error: null,
};

const riderSlice = createSlice({
  name: 'rider',
  initialState,
  reducers: {
    setDrivers(state, action: PayloadAction<Driver[]>) {
      state.drivers = action.payload;
    },
    setTripStatus(state, action: PayloadAction<TripEvents | null>) {
      state.tripStatus = action.payload;
    },
    setPaymentSession(state, action: PayloadAction<PaymentEventSessionCreatedData | null>) {
      state.paymentSession = action.payload;
    },
    setAssignedDriver(state, action: PayloadAction<Driver | null>) {
      state.assignedDriver = action.payload;
    },
    setTrip(state, action: PayloadAction<TripPreview | null>) {
      state.trip = action.payload;
    },
    setDestination(state, action: PayloadAction<[number, number] | null>) {
      state.destination = action.payload;
    },
    setError(state, action: PayloadAction<string | null>) {
      state.error = action.payload;
    },
    resetTrip(state) {
      state.tripStatus = null;
      state.paymentSession = null;
      state.trip = null;
      state.destination = null;
      state.assignedDriver = null;
    },
  },
});

export const {
  setDrivers,
  setTripStatus,
  setPaymentSession,
  setAssignedDriver,
  setTrip,
  setDestination,
  setError,
  resetTrip,
} = riderSlice.actions;

export default riderSlice.reducer;
