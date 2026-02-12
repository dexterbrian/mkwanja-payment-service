# Mkwanja Payment Service: Rules

1. **Security First:** Never log sensitive user data (like PINs or full phone numbers in plain text) in logs or audit traces.
2. **Atomic Operations:** Ensure all database updates related to financial status are atomic.
3. **Idempotency is Mandatory:** No payment initiation endpoint should function without a valid idempotency key.
4. **Idiot-Proof Callbacks:** Always assume incoming callback data could be malicious. Verify signatures and origins before updating transaction statuses.
5. **Separation of Concerns:** The payment service should not hold user balance logic; it only handles the execution and logging of the transfer. Balance is updated via DB triggers or by the App after verification.
