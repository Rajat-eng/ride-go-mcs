import { PaymentEventSessionCreatedData } from "../contracts"
import { Button } from "./ui/button"
import { loadStripe } from "@stripe/stripe-js"
import { useEffect, useState } from "react"
import { StripeConfigResponseSchema } from "../lib/schemas"

interface StripePaymentButtonProps {
  paymentSession: PaymentEventSessionCreatedData
  isLoading?: boolean
}

const formatMoney = (amount: number, currency: string) => new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: currency || 'USD',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
}).format(amount);

export const StripePaymentButton = ({
  paymentSession,
  isLoading = false,
}: StripePaymentButtonProps) => {
  const [stripePublicKey, setStripePublicKey] = useState("")
  const [isStripeConfigLoading, setIsStripeConfigLoading] = useState(true)

  useEffect(() => {
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
      console.error("Missing Stripe publishable key")
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
        Stripe publishable key is not set
      </Button>
    )
  }

  return (
    <Button
      onClick={handlePayment}
      disabled={isLoading}
      className="w-full"
    >
      {isLoading ? "Loading..." : `Pay ${formatMoney(paymentSession.amount, paymentSession.currency)}`}
    </Button>
  )
} 