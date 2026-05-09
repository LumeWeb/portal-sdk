## 0.1.52 (2026-05-09)

### Fixes

- add PausedAt field to Subscriber type

## 0.1.51 (2026-05-09)

### Fixes

- add StatusConflict to OpDeleteAccount error map

## 0.1.50 (2026-05-09)

### Features

- regenerate client code and mocks from updated specs
- add checkout session status, customer portal, and subscription events endpoints
- add WebsiteService with BlockWebsite and UnblockWebsite endpoints
- replace GetSubscriptionEvents with SSE-based SubscribeBillingEvents

### Fixes

- update swagger specs from running services

## 0.1.49 (2026-04-29)

### Features

- add billing sync APIs for pricing plans

## 0.1.48 (2026-04-29)

### Features

- update swagger specs and add DRY error sentinels

## 0.1.47 (2026-04-29)

### Fixes

- add allow_free field to pricing plan period

## 0.1.46 (2026-04-29)

### Fixes

- update admin quota spec and regenerate client

## 0.1.45 (2026-04-28)

### Fixes

- unify error handling and update swagger specs

## 0.1.44 (2026-04-28)

### Fixes

- include server error body in admin validate

## 0.1.43 (2026-04-28)

### Fixes

- expose internal billing types for consumers

## 0.1.42 (2026-04-20)

### Fixes

- export CheckoutUIFragment type from account package

## 0.1.41 (2026-04-20)

### Fixes

- add will_cancel_at field to SubscriptionStatusResponse

## 0.1.40 (2026-04-18)

### Fixes

- update ChangePlan to use period_id in request body

## 0.1.39 (2026-04-18)

### Fixes

- return PriceLineDetailResponse from UpdatePlanPosition

## 0.1.38 (2026-04-17)

### Fixes

- correct AddPlanToPriceLine response handling

## 0.1.37 (2026-04-17)

### Fixes

- convert PriceLineDetailResponse.Plans to public PricingPlanItem type

## 0.1.36 (2026-04-17)

### Features

- add PauseUserSubscription and ResumeUserSubscription billing methods

## 0.1.35 (2026-04-17)

### Features

- allow users to resume paused subscriptions

## 0.1.34 (2026-04-17)

### Features

- support pausing subscriptions via API

## 0.1.33 (2026-04-17)

### Fixes

- add immediate option for subscription cancellation

## 0.1.32 (2026-04-17)

### Features

- add abort subscription cancellation endpoints

### Fixes

- add missing 403 error handling for abort cancellation

## 0.1.31 (2026-04-16)

### Fixes

- add options pattern for GetCheckoutUI params

## 0.1.30 (2026-04-15)

### Fixes

- add missing fields to PricingPlanCreateRequest

## 0.1.29 (2026-04-15)

### Features

- add price line plan management API

## 0.1.28 (2026-04-12)

### Fixes

- resolve pricing plan request structure errors

## 0.1.27 (2026-04-12)

### Fixes

- enable external packages to create pricing plans with periods

## 0.1.26 (2026-04-12)

### Fixes

- align CreditCreateRequest with generated client types

## 0.1.25 (2026-04-12)

### Fixes

- align admin package with account pattern using generated types

## 0.1.24 (2026-04-12)

### Fixes

- configure OpenAPI overlay to use shopspring/decimal for Decimal type

## 0.1.23 (2026-04-12)

### Features

- add subscriber and subscription management APIs

## 0.1.22 (2026-04-11)

### Features

- enable price line lookup by ID

### Fixes

- correct UUID type generation from integer array to UUID string

## 0.1.21 (2026-04-11)

### Features

- add comprehensive billing API

### Fixes

- add reserved field to QuotaTypeStatus

## 0.1.20 (2026-04-07)

### Features

- add percentage-based download rate limiter

### Fixes

- correct threshold comparison in percentage rate limiter

## 0.1.19 (2026-04-06)

### Fixes

- add AllowDownload method to RateLimiterFunc

## 0.1.18 (2026-04-06)

### Features

- add download rate limiter for SDK integration

### Fixes

- validate non-negative size in rate limiter

## 0.1.17 (2026-03-30)

### Fixes

- update swagger specs and regenerate client code

## 0.1.16 (2026-03-29)

### Fixes

- update swagger specs and regenerate client code

## 0.1.15 (2026-03-28)

### Features

- add user quota config APIs

### Fixes

- update admin quota spec with 204 response documentation
- remove contradictory 200 responses from all DELETE operations

## 0.1.14 (2026-03-26)

### Fixes

- update quota response from bandwidth to storage field
- update GetQuotaHistory usage type from bandwidth to storage

## 0.1.13 (2026-03-26)

### Fixes

- align response handling with updated API specification

## 0.1.12 (2026-03-24)

### Features

- implement multi-spec admin API with quota service

### Fixes

- address PR review feedback
- remove unused userID field from ListAllowances test

## 0.1.11 (2026-03-09)

### Features

- add profile management endpoints to AccountAPI

### Fixes

- address PR review feedback

## 0.1.10 (2026-03-08)

### Fixes

- update OTP auth cookie and query param names

## 0.1.9 (2026-03-08)

### Fixes

- add runtime redirect control for OTP validate endpoint

## 0.1.8 (2026-03-08)

### Fixes

- update swagger spec and regenerated client for OTP validate endpoint

## 0.1.7 (2026-03-06)

### Features

- add host override support for vhost testing

### Fixes

- address review feedback from Kody code review

## 0.1.6 (2026-03-06)

### Features

- add password reset and update APIs
- add ResendVerifyEmail method

## 0.1.5 (2026-03-06)

### Features

- add DeleteAccount method to AccountAPI

### Fixes

- add error handling for Hijack() in network error tests

## 0.1.4 (2026-02-14)

### Fixes

- update mocks

## 0.1.3 (2026-02-14)

### Features

- add API key login support

## 0.1.2 (2026-02-03)

### Fixes

- need to override BinaryUUID go type

## 0.1.1 (2026-01-07)

### Features

- add public constructors for APIKey and LoginResult

### Fixes

- add public constructors for APIKey and LoginResult

## 0.1.0 (2026-01-05)

### Breaking Changes

- Initial release

### Features

- initial implementation of the Account SDK

### Fixes

- use url.Parse for proper token extraction from Location header
- correct DeleteAPIKey parameter type to use string UUID
