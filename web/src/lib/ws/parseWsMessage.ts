/**
 * parseWsMessage
 *
 * Safely parses a raw WebSocket frame into a typed server message.
 *
 * Returns a discriminated result so callers can handle validation
 * failures without throwing:
 *
 *   const result = parseWsMessage(rawEvent.data);
 *   if (!result.ok) { dispatch(addError(result.error)); return; }
 *   switch (result.message.type) { ... }
 */
import { ServerWsMessageSchema, ParsedServerWsMessage } from '../schemas/ws.server.schemas';
import { ClientWsMessageSchema, ValidatedClientWsMessage } from '../schemas/ws.client.schemas';

// ─── Result types ─────────────────────────────────────────────────────────────

export type WsParseSuccess = { ok: true; message: ParsedServerWsMessage };
export type WsParseFailure = { ok: false; error: string; raw: unknown };
export type WsParseResult = WsParseSuccess | WsParseFailure;

// ─── Inbound parse ────────────────────────────────────────────────────────────

/**
 * Parses a raw string frame from the WebSocket onmessage handler.
 * Returns a typed discriminated result — never throws.
 */
export function parseWsMessage(rawFrame: string): WsParseResult {
  let json: unknown;

  try {
    json = JSON.parse(rawFrame);
  } catch {
    return {
      ok: false,
      error: `WebSocket frame is not valid JSON: ${rawFrame.slice(0, 120)}`,
      raw: rawFrame,
    };
  }

  const result = ServerWsMessageSchema.safeParse(json);

  if (!result.success) {
    const issues = result.error.issues
      .map((i) => `[${i.path.join('.')}] ${i.message}`)
      .join('; ');

    return {
      ok: false,
      error: `WS message validation failed — ${issues}`,
      raw: json,
    };
  }

  return { ok: true, message: result.data };
}

// ─── Outbound validation ──────────────────────────────────────────────────────

export type ClientValidateSuccess = { ok: true; payload: ValidatedClientWsMessage };
export type ClientValidateFailure = { ok: false; error: string };
export type ClientValidateResult = ClientValidateSuccess | ClientValidateFailure;

/**
 * Validates an outgoing client message before sending.
 * Returns { ok: false } with a human-readable error string on failure.
 */
export function validateClientMessage(message: unknown): ClientValidateResult {
  const result = ClientWsMessageSchema.safeParse(message);

  if (!result.success) {
    const issues = result.error.issues
      .map((i) => `[${i.path.join('.')}] ${i.message}`)
      .join('; ');

    return { ok: false, error: `Outgoing WS message invalid — ${issues}` };
  }

  return { ok: true, payload: result.data };
}
