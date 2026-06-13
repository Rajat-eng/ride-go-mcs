import { useCallback, useRef } from 'react';
import { useAppDispatch, useAppSelector } from '../store/store';
import { setTrip, setDestination, resetTrip, setTripStatus } from '../store/slices/riderSlice';
import { RouteFare } from '../types';
import { HTTPTripPreviewRequestPayload, TripEvents } from '../contracts';
import { usePreviewTripMutation, useStartTripMutation, useCancelTripMutation } from '../store/api/tripApi';
import { TripPreviewApiResponseSchema, StartTripResponseSchema } from '../lib/schemas';

export function useRiderTrip(userID: string) {
  const dispatch = useAppDispatch();
  const { trip, destination } = useAppSelector((s) => s.rider);
  const debounceTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const [previewTrip] = usePreviewTripMutation();
  const [startTrip] = useStartTripMutation();
  const [cancelTrip] = useCancelTripMutation();

  const handleMapClick = useCallback(async (
    latlng: { lat: number; lng: number },
    pickupLocation: { latitude: number; longitude: number },
    onRouteSelected?: (distance: number) => void,
  ) => {
    if (trip?.tripID) return;

    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current);
    }

    debounceTimeoutRef.current = setTimeout(async () => {
      dispatch(setDestination([latlng.lat, latlng.lng]));

      const payload: HTTPTripPreviewRequestPayload = {
        userID,
        pickup: { latitude: pickupLocation.latitude, longitude: pickupLocation.longitude },
        destination: { latitude: latlng.lat, longitude: latlng.lng },
      };

      const raw = await previewTrip(payload).unwrap();
      // Prefer validated shape; fall back to raw cast so a schema mismatch
      // never silently erases the destination pin the user just placed.
      const parsed = TripPreviewApiResponseSchema.safeParse(raw);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const data = parsed.success ? parsed.data.data : (raw as any)?.data ?? raw;

      if (!data?.route) {
        dispatch(setDestination(null));
        return;
      }

      const parsedRoute = data.route.geometry[0].coordinates
        .map((coord: { longitude: number; latitude: number }) => [coord.longitude, coord.latitude] as [number, number]);

      dispatch(setTrip({
        tripID: "",
        route: parsedRoute,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        rideFares: data.rideFares as any,
        distance: data.route.distance,
        duration: data.route.duration,
      }));

      onRouteSelected?.(data.route.distance);
    }, 500);
  }, [trip?.tripID, dispatch, previewTrip, userID]);

  const handleStartTrip = useCallback(async (fare: RouteFare) => {
    // called when user clicks "Confirm" on the trip overview screen
    if (!fare.id) {
      return;
    }

    const raw = await startTrip({ rideFareID: fare.id, userID }).unwrap();
    const parsed = StartTripResponseSchema.safeParse(raw);
    const data = parsed.success ? parsed.data : raw as { tripID: string };

    if (trip) {
      dispatch(setTrip({ ...trip, tripID: data.tripID, selectedFare: fare }));
    }
    dispatch(setTripStatus(TripEvents.Created));

    return data;
  }, [userID, trip, dispatch, startTrip]);

  const handleCancelTrip = useCallback(async () => {
    const tripID = trip?.tripID;
    if (tripID) {
      try {
        await cancelTrip({ tripID }).unwrap();
      } catch {
        // Even if the server call fails (e.g. trip already cancelled), reset local state.
      }
    }
    dispatch(resetTrip());
  }, [trip?.tripID, cancelTrip, dispatch]);

  return {
    trip,
    destination,
    handleMapClick,
    handleStartTrip,
    handleCancelTrip,
  };
}
