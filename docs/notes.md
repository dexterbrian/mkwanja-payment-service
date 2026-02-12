# Mkwanja Payment Service: Notes

## Technical Highlights
- **Performance:** Fiber v2 is chosen for its performance and low memory footprint, essential for high-frequency financial webhooks.
- **Idempotency:** A critical v1 feature. Using client-side UUIDs ensures that network retries don't result in double-spending.

## Critical Considerations
- **Signature Verification:** We must accurately verify Daraja callbacks to prevent spoofing. Daraja typically uses IP whitelisting and payload verification.
- **Redis Availability:** Redis is used for idempotency. If Redis is down, the service should fail safe or fall back to a slower PostgreSQL-based check to prevent duplicate payments.
- **M-PESA Downtime:** The service must handle gateway timeouts gracefully and inform the user via real-time updates through Supabase.

## Future Plans (non-MVP)
- **Multi-region deployment:** For reduced latency.
- **Enhanced Fraud Detection:** Analyzing transaction patterns before sending to M-PESA.
