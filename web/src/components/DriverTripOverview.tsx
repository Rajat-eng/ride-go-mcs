import { Trip } from "../types"
import { TripOverviewCard } from "./TripOverviewCard"
import { Button } from "./ui/button"
import { ChatMessageData, TripEvents } from "../contracts"
import { TripChatPanel } from "./TripChatPanel"
import { PackagesMeta } from "./PackagesMeta"

const getRouteEndpoints = (trip: Trip | null) => {
  const coordinates = trip?.route?.geometry?.[0]?.coordinates ?? [];
  return {
    start: coordinates[0],
    destination: coordinates[coordinates.length - 1],
  };
};

interface DriverTripOverviewProps {
  trip?: Trip | null,
  status?: TripEvents | null,
  userID: string,
  chatMessages: ChatMessageData[],
  onSendChatMessage: (tripID: string, text: string) => void,
  onAcceptTrip?: () => void,
  onDeclineTrip?: () => void,
  onCancelTrip?: () => void,
}

export const DriverTripOverview = ({ trip, status, userID, chatMessages, onSendChatMessage, onAcceptTrip, onDeclineTrip, onCancelTrip }: DriverTripOverviewProps) => {
  const { start, destination } = getRouteEndpoints(trip ?? null);

  if (!trip) {
    return (
      <TripOverviewCard
        title="Waiting for a rider..."
        description="Waiting for a rider to request a trip..."
      />
    )
  }

  if (status === TripEvents.DriverTripRequest) {
    return (
      <TripOverviewCard
        title="Trip request received!"
        description="A trip has been requested, check the route and accept the trip if you can take it."
      >
        <div className="flex flex-col gap-2 text-sm text-gray-600 mb-3">
          <p>Trip ID: {trip.id}</p>
          <p>Rider: {trip.userID}</p>
          {trip?.selectedFare?.packageSlug && (
            <p>Package selected: {PackagesMeta[trip.selectedFare.packageSlug].name} ({trip.selectedFare.packageSlug})</p>
          )}
          {start && destination && (
            <p>
              Start: {start.latitude.toFixed(5)}, {start.longitude.toFixed(5)}
              <br />
              Destination: {destination.latitude.toFixed(5)}, {destination.longitude.toFixed(5)}
            </p>
          )}
        </div>
        <div className="flex flex-col gap-2">
          <Button onClick={onAcceptTrip}>Accept trip</Button>
          <Button variant="outline" onClick={onDeclineTrip}>Decline trip</Button>
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.DriverTripAccept) {
    return (
      <TripOverviewCard
        title="All set!"
        description="You can now start the trip"
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <h3 className="text-lg font-bold">Trip details</h3>
            <p className="text-sm text-gray-500">
              Trip ID: {trip.id}
              <br />
              Rider ID: {trip.userID}
              {trip?.selectedFare?.packageSlug && (
                <>
                  <br />
                  Package: {PackagesMeta[trip.selectedFare.packageSlug].name} ({trip.selectedFare.packageSlug})
                </>
              )}
            </p>
          </div>
          <TripChatPanel
            title="Chat with rider"
            tripID={trip.id}
            currentUserID={userID}
            peerLabel={trip.userID}
            messages={chatMessages}
            onSend={(text) => onSendChatMessage(trip.id, text)}
          />
          <Button variant="destructive" onClick={onCancelTrip}>Cancel trip</Button>
        </div>
      </TripOverviewCard>
    )
  }

  return null
}