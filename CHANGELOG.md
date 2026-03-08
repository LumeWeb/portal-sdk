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
