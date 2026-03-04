# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 0.1.x   | Yes |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in OpenPrint Cloud, please report it responsibly.

### How to Report

**DO NOT** create a public GitHub issue for security vulnerabilities.

Instead, please:

1. Email security@openprint.ai with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Affected versions
   - Proof of concept (if available)

2. You will receive a response within **48 hours** acknowledging receipt

3. We will investigate and keep you informed of progress

### What to Include

When reporting, please include:

- **Vulnerability Type**: SQL injection, XSS, CSRF, authentication bypass, etc.
- **Attack Vector**: How the vulnerability can be exploited
- **Scope**: What data/functionality is at risk
- **Impact**: Severity of potential damage
- **Reproduction**: Step-by-step guide to reproduce
- **Proof of Concept**: Working exploit code (optional but helpful)

## Security Measures

### Authentication & Authorization

#### JWT Tokens
- **Algorithm**: HS256 (HMAC-SHA256)
- **Token Expiry**: Access tokens expire after 24 hours
- **Refresh Tokens**: Single-use, invalidated after use
- **Secret Key**: Minimum 32 characters required

#### Password Security
- **Hashing**: bcrypt with cost factor 12
- **Requirements**: Minimum 8 characters
- **Validation**: Common password blacklist
- **Reset**: Time-limited tokens for password reset

#### Session Management
- **Storage**: Redis with encryption at rest
- **Timeout**: 24-hour inactivity timeout
- **Concurrent Sessions**: Limited to 5 per user

### Data Protection

#### Encryption
- **At Rest**: AES-256 encryption for sensitive data
- **In Transit**: TLS 1.3+ required for all connections
- **Key Management**: Environment variables, never in code
- **Rotation**: Encryption keys can be rotated

#### Secrets Management
- **Storage**: Environment variables or secret management system
- **Logging**: Secrets never logged or exposed in errors
- **Git**: Secrets never committed to repository
- **Access**: Principle of least privilege

### API Security

#### Rate Limiting
- **Global**: 10,000 requests per hour
- **Per IP**: 100 requests per minute
- **Per User**: 1,000 requests per hour
- **Per API Key**: 5,000 requests per hour
- **Auth Endpoints**: Stricter limits (10 per minute)

#### Input Validation
- **All Inputs**: Validated on server-side
- **SQL Injection**: Parameterized queries only
- **XSS**: Output encoding and CSP headers
- **CSRF**: Token-based protection for state-changing operations

#### CORS Policy
- **Origins**: Whitelist-based
- **Methods**: Specific methods only (no wildcard)
- **Credentials**: Enabled for authenticated requests
- **Headers**: Restricted to necessary headers

### Infrastructure Security

#### Network
- **Firewall**: Restricted port access
- **TLS**: Required for all external communication
- **HSTS**: Strict Transport Security enabled
- **Certificate**: Valid TLS certificates only

#### Container Security
- **Base Image**: Alpine Linux (minimal attack surface)
- **User**: Non-root user in containers
- **Read-only**: Filesystem where possible
- **Capabilities**: Dropped unnecessary capabilities

#### Database
- **Encryption**: TLS for connections
- **Access**: Limited user permissions
- **Backups**: Encrypted backups
- **Audit**: Query logging for sensitive operations

## Security Features

### Multi-Factor Authentication (MFA)
- **TOTP**: Time-based one-time passwords
- **Backup Codes**: Recovery codes
- **Enforcement**: Configurable per organization

### Audit Logging
- **Events**: All sensitive operations logged
- **Retention**: 90 days default
- **Storage**: Immutable audit trail
- **Access**: Restricted to administrators

### Intrusion Detection
- **Rate Limiting**: DDoS protection
- **Brute Force**: Account lockout after failed attempts
- **Anomaly Detection**: Unusual pattern detection
- **Alerting**: Real-time security alerts

### Secure Print Release
- **Authentication**: User must authenticate at printer
- **Timeout**: Jobs held for configurable period
- **Tracking**: All release events logged
- **Cancellation**: User can cancel their own jobs

## Security Headers

All HTTP responses include:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' 'self'; object-src 'self'
```

## Security Best Practices

### For Developers
1. **Never commit secrets** to repository
2. **Validate all inputs** on server-side
3. **Use parameterized queries** for database operations
4. **Keep dependencies updated** for security patches
5. **Follow principle of least privilege**
6. **Log security events** but not sensitive data
7. **Review code** for security issues before merging

### For Administrators
1. **Use strong JWT secrets** (32+ characters)
2. **Enable MFA** for all admin accounts
3. **Monitor audit logs** regularly
4. **Keep backups encrypted** and secure
5. **Update regularly** for security patches
6. **Review access controls** periodically
7. **Test incident response** procedures

### For Users
1. **Use strong passwords** (8+ characters, mix of types)
2. **Enable MFA** when available
3. **Report suspicious activity** immediately
4. **Don't share credentials** with others
5. **Log out** when finished
6. **Verify URLs** before clicking links
7. **Keep software updated** in your browser

## Compliance

OpenPrint Cloud is designed to support compliance with:

- **FedRAMP**: Federal Risk and Authorization Management Program
- **HIPAA**: Health Insurance Portability and Accountability Act
- **GDPR**: General Data Protection Regulation
- **SOC2**: Service Organization Control 2

Contact compliance@openprint.ai for compliance-related questions.

## Security Updates

Security updates are released:
- **Patch Releases**: As needed for critical issues
- **Minor Releases**: Monthly with security improvements
- **Major Releases**: Quarterly with security enhancements

Subscribe to security announcements at security-announce@openprint.ai

## Vulnerability Disclosure Program

We appreciate responsible disclosure and offer:

- **Recognition**: Credit in our security hall of fame
- **Timeline**: 90 days to fix before public disclosure
- **Communication**: Regular updates during fix process
- **Bounty**: Consideration for critical vulnerabilities (case-by-case)

## Contact

- **Security Issues**: security@openprint.ai
- **General Questions**: security-questions@openprint.ai
- **Compliance**: compliance@openprint.ai
- **PGP Key**: Available at https://openprint.ai/security.asc

Thank you for helping keep OpenPrint Cloud secure! 🔒
