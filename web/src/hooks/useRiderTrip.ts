import { useCallback, useRef } from 'react';
import { useAppDispatch, useAppSelector } from '../store/store';
import { setTrip, setDestination, resetTrip } from '../store/slices/riderSlice';
import { RouteFare } from '../types';
import { HTTPTripPreviewRequestPayload } from '../contracts';
import { usePreviewTripMutation, useStartTripMutation } from '../store/api/tripApi';

export function useRiderTrip(userID: string) {
  const dispatch = useAppDispatch();
  const { trip, destination } = useAppSelector((s) => s.rider);
  const debounceTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const [previewTrip] = usePreviewTripMutation();
  const [startTrip] = useStartTripMutation();

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

      const result = await previewTrip(payload).unwrap();
      const data = result.data;

      if (data.route == null) {
        dispatch(setDestination(null));
        return;
      }

      const parsedRoute = data.route.geometry[0].coordinates
        .map((coord) => [coord.longitude, coord.latitude] as [number, number]);

      dispatch(setTrip({
        tripID: "",
        route: parsedRoute,
        rideFares: data.rideFares,
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

    const data = await startTrip({ rideFareID: fare.id, userID }).unwrap();

    if (trip) {
      dispatch(setTrip({ ...trip, tripID: data.tripID, selectedFare: fare }));
    }

    return data;
  }, [userID, trip, dispatch, startTrip]);

  const handleCancelTrip = useCallback(() => {
    dispatch(resetTrip());
  }, [dispatch]);

  return {
    trip,
    destination,
    handleMapClick,
    handleStartTrip,
    handleCancelTrip,
  };
}
