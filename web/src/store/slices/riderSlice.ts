import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Driver, TripPreview } from '../../types';
import { ChatMessageData, PaymentEventSessionCreatedData, TripEvents } from '../../contracts';

import { Coordinate } from '../../types';

interface RiderState {
  ownerUserID: string | null;
  drivers: Driver[];
  tripStatus: TripEvents | null;
  paymentSession: PaymentEventSessionCreatedData | null;
  assignedDriver: Driver | null;
  assignedDriverLocation: Coordinate | null;
  chatMessages: ChatMessageData[];
  trip: TripPreview | null;
  destination: [number, number] | null;
  error: string | null;
}

const initialState: RiderState = {
  ownerUserID: null,
  drivers: [],
  tripStatus: null,
  paymentSession: null,
  assignedDriver: null,
  assignedDriverLocation: null,
  chatMessages: [],
  trip: null,
  destination: null,
  error: null,
};

const riderSlice = createSlice({
  name: 'rider',
  initialState,
  reducers: {
    setOwnerUserID(state, action: PayloadAction<string | null>) {
      state.ownerUserID = action.payload;
    },
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
    setAssignedDriverLocation(state, action: PayloadAction<Coordinate | null>) {
      state.assignedDriverLocation = action.payload;
    },
    addChatMessage(state, action: PayloadAction<ChatMessageData>) {
      state.chatMessages.push(action.payload);
    },
    setError(state, action: PayloadAction<string | null>) {
      state.error = action.payload;
    },
    completeTrip(state) {
      state.tripStatus = TripEvents.Completed;
      state.paymentSession = null;
      state.trip = null;
      state.destination = null;
      state.assignedDriver = null;
      state.assignedDriverLocation = null;
      state.chatMessages = [];
      state.drivers = [];
    },
    resetTrip(state) {
      state.tripStatus = null;
      state.paymentSession = null;
      state.trip = null;
      state.destination = null;
      state.assignedDriver = null;
      state.assignedDriverLocation = null;
      state.chatMessages = [];
    },
    clearState() {
      return initialState;
    },
  },
});

export const {
  setOwnerUserID,
  setDrivers,
  setTripStatus,
  setPaymentSession,
  setAssignedDriver,
  setAssignedDriverLocation,
  addChatMessage,
  setTrip,
  setDestination,
  setError,
  completeTrip,
  resetTrip,
  clearState,
} = riderSlice.actions;

export default riderSlice.reducer;
