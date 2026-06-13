import { PaymentEventSessionCreatedData } from "../contracts"
import { Button } from "./ui/button"
import { loadStripe } from "@stripe/stripe-js"
import { useEffect, useState } from "react"
import { StripeConfigResponseSchema } from "../lib/schemas"

interface StripePaymentButtonProps {
  paymentSession: PaymentEventSessionCreatedData
  isLoading?: boolean
}

const buildTimeStripePublicKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? ""

export const StripePaymentButton = ({
  paymentSession,
  isLoading = false,
}: StripePaymentButtonProps) => {
  const [stripePublicKey, setStripePublicKey] = useState(buildTimeStripePublicKey)
  const [isStripeConfigLoading, setIsStripeConfigLoading] = useState(!buildTimeStripePublicKey)

  useEffect(() => {
    if (buildTimeStripePublicKey) {
      return
    }

    let isMounted = true

    const loadStripeConfig = async () => {
      try {
        const response = await fetch("/api/config/stripe", {
          cache: "no-store",
        })

        if (!response.ok) {
          throw new Error(`Failed to load Stripe config: ${response.status}`)
        }

        const raw = await response.json()
        const parsed = StripeConfigResponseSchema.safeParse(raw)

        if (isMounted) {
          setStripePublicKey(parsed.success ? parsed.data.publishableKey : "")
        }
      } catch (error) {
        console.error("Unable to load Stripe publishable key", error)
      } finally {
        if (isMounted) {
          setIsStripeConfigLoading(false)
        }
      }
    }

    void loadStripeConfig()

    return () => {
      isMounted = false
    }
  }, [])

  const handlePayment = async () => {
    if (!stripePublicKey) {
      console.error("Missing NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY")
      return
    }

    const stripe = await loadStripe(stripePublicKey)

    if (!stripe) {
      console.error("Stripe failed to load")
      return
    }

    // Redirect to Stripe Checkout
    const { error } = await stripe.redirectToCheckout({ sessionId: paymentSession.sessionID })

    if (error) {
      console.error("Payment error:", error)
    }
  }

  if (isStripeConfigLoading) {
    return (
      <Button
        disabled
        className="w-full"
      >
        Loading payment...
      </Button>
    )
  }

  if (!stripePublicKey) {
    return (
      <Button
        disabled
        className="w-full bg-red-500 text-white"
      >
        NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY is not set
      </Button>
    )
  }

  return (
    <Button
      onClick={handlePayment}
      disabled={isLoading}
      className="w-full"
    >
      {isLoading ? "Loading..." : `Pay ${paymentSession.amount} ${paymentSession.currency}`}
    </Button>
  )
} 