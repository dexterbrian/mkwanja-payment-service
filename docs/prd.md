# Product Requirements Document: Mkwanja Payment Service

## 1. Product Overview

**Product Name:** Mkwanja Payment Service  
**Version:** 1.0 (MVP)  
**Status:** Requirements Finalization

### 1.1 Product Vision
The Mkwanja Payment Service is a secure, high-performance Go-based microservice responsible for orchestrating financial transactions between the Mkwanja ecosystem and external payment gateways (primarily M-PESA Daraja). It ensures that all money movement is authenticated, idempotent, and reliably logged.

### 1.2 Problem Statement
Handling financial transactions directly in a mobile app or a general-purpose backend (like Supabase) introduces security risks and complexity. A dedicated payment service decouples sensitive financial logic, providing a robust layer for signature verification, Daraja API integrations, and idempotency control.

---

## 2. Technical Stack

- **Language:** Go (Golang)
- **Framework:** Fiber (v2)
- **Database (Internal):** Redis (for idempotency caching) and PostgreSQL (via Supabase) for transaction logs.
- **Gateway:** M-PESA Daraja API (STK Push, B2C, B2B).
- **Auth:** JWT-based verification using Supabase signing keys.

---

## 3. MVP Features

### 3.1 M-PESA Integration
- **STK Push (Lipan na M-PESA Express):** Facilitate customer deposits into the Mkwanja ecosystem.
- **B2C (Business to Customer):** Process withdrawals from Mkwanja to personal M-PESA accounts.
- **B2B (Business to Business):** Execute utility payments (Paybill/Till) for budget items like Rent, KPLC, etc.

### 3.2 Idempotency (v1)
- **Requirement:** Every transaction initiation request must include a unique `X-Idempotency-Key`.
- **Logic:** If a request with an existing key is received within a 24-hour window, the service must return the original response without re-executing the payment.
- **Storage:** Use Redis or a dedicated PostgreSQL table for high-speed lookups of handled keys.

### 3.3 Security & Verification
- **JWT Authentication:** All incoming requests from the Flutter App or Supabase Edge Functions must be authenticated via JWT.
- **M-PESA Callback Verification:** Verify Daraja API signatures and IP addresses for all incoming callbacks to prevent spoofing.

### 3.4 Transaction Logging
- Real-time logging of transaction status (Pending, Success, Failed) to the central PostgreSQL database.
- Storage of raw callback data for audit trails and debugging.

---

## 4. System Architecture

The Payment Service acts as an intermediary:
1. **Request:** App/Edge Function sends a payment request with JWT and Idempotency Key.
2. **Authorize:** Service validates JWT.
3. **Check Idempotency:** Service checks if the key was already processed.
4. **Execute:** Service calls M-PESA Daraja API.
5. **Log:** Service logs the initial "Pending" state in the database.
6. **Callback:** External gateway calls the service's callback URL.
7. **Finalize:** Service validates callback, updates database to "Success/Failed", and triggers a real-time event.
