import { NextResponse } from "next/server"

export const dynamic = "force-dynamic"

export async function GET() {
  return NextResponse.json(
    {
      publishableKey:
        process.env.STRIPE_PUBLISHABLE_KEY ??
        process.env["NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY"] ??
        "",
    },
    {
      headers: {
        "Cache-Control": "no-store",
      },
    },
  )
}