/**
 * Barrel export for all Zod schemas.
 *
 * Import from here rather than from individual schema files
 * so that reorganising the internals is a one-line change.
 *
 * @example
 *   import { ServerWsMessageSchema, AuthUserSchema } from '@/lib/schemas';
 */
export * from './domain.schemas';
export * from './ws.server.schemas';
export * from './ws.client.schemas';
export * from './http.schemas';
export * from './auth.schemas';
