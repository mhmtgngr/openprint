# Cookie SameSite Mode Usage Guidelines

## Overview

This document provides guidelines for using SameSite cookie attributes in the OpenPrint application to prevent Cross-Site Request Forgery (CSRF) attacks while maintaining proper functionality.

## What is SameSite?

The SameSite attribute controls whether cookies are sent with cross-site requests. This is a critical security feature that helps prevent CSRF attacks.

### SameSite Modes

| Mode | Description | When to Use |
|------|-------------|-------------|
| **Strict** | Cookie is only sent for first-party requests (same-site). Never sent on cross-site navigation. | Highest security; auth cookies in production |
| **Lax** | Cookie is sent on same-site requests and with "safe" top-level navigations (GET requests). | Balance of security and usability; general use |
| **None** | Cookie is sent on all requests (cross-site included). Requires Secure flag. | Required for cross-origin scenarios; rarely used |

## OpenPrint Configuration

### Development Environment

```go
// DevelopmentCookieSecurity returns relaxed settings for local development
func DevelopmentCookieSecurity() *CookieSecurityConfig {
    return &CookieSecurityConfig{
        Secure:   false,  // Allow HTTP (not HTTPS)
        HttpOnly: true,   // Prevent JavaScript access
        SameSite: http.SameSiteLaxMode,  // Allow safe navigations
        Path:     "/",
    }
}
```

**Rationale:**
- `Secure: false` - Allows testing on HTTP (localhost)
- `HttpOnly: true` - Always prevents XSS from stealing cookies
- `SameSite: Lax` - Balances security with developer experience

### Production Environment

```go
// ProductionCookieSecurity returns strict settings for production deployment
func ProductionCookieSecurity() *CookieSecurityConfig {
    return &CookieSecurityConfig{
        Secure:   true,  // Require HTTPS only
        HttpOnly: true,  // Prevent JavaScript access
        SameSite: http.SameSiteStrictMode,  // Maximum CSRF protection
        Path:     "/",
    }
}
```

**Rationale:**
- `Secure: true` - Cookies only sent over HTTPS
- `HttpOnly: true` - Mitigates XSS token theft
- `SameSite: Strict` - Maximum CSRF protection

### Default (Auto-Detect)

```go
// AutoCookieSecurity automatically selects settings based on environment
func AutoCookieSecurity() *CookieSecurityConfig {
    if EnvIsProduction() {
        return ProductionCookieSecurity()
    }
    return DevelopmentCookieSecurity()
}
```

## Environment Detection

The application detects production environment via:

1. `ENV=production` environment variable
2. `GO_ENV=production` environment variable
3. Case-insensitive matching (e.g., "Production", "PRODUCTION")

```bash
# Set production mode
export ENV=production
```

## Session Cookie Management

### Setting Session Cookies

```go
import "time"

// Set a session cookie with proper security
SetSessionCookie(
    responseWriter,
    "session_token",
    tokenValue,
    15 * time.Minute,  // 15 minute expiration
    ProductionCookieSecurity(),  // Use production security
)
```

### Clearing Session Cookies

```go
// Clear session cookie (maintains security attributes)
ClearSessionCookie(
    responseWriter,
    "session_token",
    ProductionCookieSecurity(),
)
```

## Migration Guide

### Migrating from No SameSite to Lax/Strict

**Before (insecure):**
```go
http.SetCookie(w, &http.Cookie{
    Name:  "session",
    Value: token,
})
```

**After (secure):**
```go
security := AutoCookieSecurity()
SetSessionCookie(w, "session", token, 15*time.Minute, security)
```

### Handling Third-Party Integrations

For scenarios requiring cross-site cookie handling (e.g., OAuth callbacks):

1. **Use SameSite=None with caution:**
   ```go
   &CookieSecurityConfig{
       Secure:   true,  // REQUIRED with SameSite=None
       HttpOnly: true,
       SameSite: http.SameSiteNoneMode,
   }
   ```

2. **Consider state-based flow instead** (preferred for OAuth)

## Testing

### Unit Tests

Cookie security settings are tested in:
- `internal/middleware/cookie_auth_test.go`

### Testing Locally

```bash
# Development (HTTP, Lax mode)
ENV=development go run ./cmd/auth-service

# Production simulation (HTTPS, Strict mode)
ENV=production go run ./cmd/auth-service
```

## Browser Compatibility

| Browser | SameSite=Strict | SameSite=Lax | SameSite=None |
|---------|-----------------|--------------|---------------|
| Chrome 51+ | ✅ | ✅ | ✅ |
| Firefox 60+ | ✅ | ✅ | ✅ |
| Safari 12+ | ✅ | ✅ | ✅ |
| Edge 79+ | ✅ | ✅ | ✅ |
| IE 11 | ❌ | ❌ | ❌ |

**Note:** OpenPrint requires modern browsers with SameSite support.

## Security Considerations

### CSRF Protection

SameSite cookies are one layer of CSRF protection. OpenPrint also uses:

1. CSRF tokens for state-changing operations
2. Same-origin policy enforcement
3. Origin and Referer header validation

### XSS Protection

`HttpOnly` flag is **always** set to `true` regardless of environment to prevent JavaScript access to session cookies.

### Cookie Expiration

- Access tokens: 15 minutes (short-lived)
- Refresh tokens: 48 hours (configurable, max 7 days)

## Troubleshooting

### Issue: Login Redirect Loops

**Cause:** SameSite=Strict blocking OAuth redirects

**Solution:**
```go
// Use Lax mode for OAuth flows
security := &CookieSecurityConfig{
    Secure:   true,
    HttpOnly: true,
    SameSite: http.SameSiteLaxMode,
}
```

### Issue: Cookies Not Set on HTTPS

**Cause:** Missing `Secure` flag

**Solution:** Use `ProductionCookieSecurity()` or ensure `Secure: true`

### Issue: Third-Party Embed Not Working

**Cause:** SameSite blocking cross-origin iframe requests

**Solution:** Use `SameSite=None` with `Secure: true` (if absolutely necessary)

## Best Practices

1. **Always use HttpOnly cookies** - Never allow JavaScript access to session tokens
2. **Use Strict mode in production** - Maximum security for authentication cookies
3. **Test in development first** - Use Lax mode during development to avoid frustration
4. **Log environment detection** - Help debug which mode is active
5. **Never use SameSite=None** without Secure flag - Browsers will reject the cookie
6. **Monitor browser changes** - SameSite behavior evolves with browser updates

## References

- [MDN: SameSite cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite)
- [OWASP: CSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [RFC 6265bis: SameSite Cookies](https://datatracker.ietf.org/doc/html/draft-ietf-httpbis-rfc6265bis)
