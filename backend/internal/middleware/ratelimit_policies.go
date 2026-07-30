package middleware

import "time"

// Named rate-limit policies for every category of endpoint that needs
// its own budget. Unauthenticated, credential-guessing-shaped endpoints
// (login, password reset, verification) get tight IP limits; endpoints
// that are expensive to run (AI) get per-user *and* per-org ceilings so
// cost is bounded per tenant regardless of how many seats they have; and
// a broad per-user safety net sits under the whole authenticated API so
// no single route needs its own bespoke limiter just to have *a* limit.

// RateLimitLogin throttles login attempts per IP. Deliberately tighter
// than register/refresh — this is the endpoint credential-stuffing and
// brute-force tools hit hardest, and it's on top of the per-account
// lockout in AuthService (which stops one *account* being brute-forced;
// this stops one *IP* trying many accounts).
func RateLimitLogin() RateLimitConfig {
	return RateLimitConfig{Name: "login", Limit: 20, Window: 5 * time.Minute, Burst: 6, BurstWindow: 10 * time.Second, KeyFunc: ByIP}
}

// RateLimitLoginByAccount catches distributed credential stuffing —
// many IPs, one targeted account — that RateLimitLogin's IP-keyed limit
// can't see. Looser than the per-account lockout in AuthService (which
// already locks the account after a handful of failures); this exists
// specifically so an attacker spreading attempts across IPs to dodge
// RateLimitLogin still hits a ceiling tied to the account they're after.
func RateLimitLoginByAccount() RateLimitConfig {
	return RateLimitConfig{Name: "login_account", Limit: 15, Window: 15 * time.Minute, KeyFunc: ByBodyEmail}
}

func RateLimitPasswordResetByAccount() RateLimitConfig {
	return RateLimitConfig{Name: "password_reset_account", Limit: 5, Window: time.Hour, KeyFunc: ByBodyEmail}
}

func RateLimitEmailVerificationByAccount() RateLimitConfig {
	return RateLimitConfig{Name: "email_verification_account", Limit: 5, Window: time.Hour, KeyFunc: ByBodyEmail}
}

func RateLimitMFAVerify() RateLimitConfig {
	return RateLimitConfig{Name: "mfa_verify", Limit: 15, Window: 5 * time.Minute, Burst: 5, BurstWindow: 10 * time.Second, KeyFunc: ByIP}
}

// RateLimitMFAManage covers the authenticated MFA confirm/disable
// endpoints, both of which are effectively "guess a 6-digit code" or
// "guess my own password" surfaces once a session is already
// hijacked/shared — the broad RateLimitAPI safety net (600/min) is far
// too loose to meaningfully slow either kind of guessing down.
func RateLimitMFAManage() RateLimitConfig {
	return RateLimitConfig{Name: "mfa_manage", Limit: 10, Window: 5 * time.Minute, KeyFunc: ByUser}
}

func RateLimitRegister() RateLimitConfig {
	return RateLimitConfig{Name: "register", Limit: 5, Window: time.Hour, KeyFunc: ByIP}
}

// RateLimitOrgLookup covers the public "does this organization exist"
// check the signup form calls on every keystroke-settled org name. Loose
// (this leaks only existence + display name/slug, already by design —
// see OrgLookupResult) but still bounded so it can't be used to
// enumerate the whole organization namespace at will.
func RateLimitOrgLookup() RateLimitConfig {
	return RateLimitConfig{Name: "org_lookup", Limit: 30, Window: 5 * time.Minute, Burst: 10, BurstWindow: 10 * time.Second, KeyFunc: ByIP}
}

// RateLimitJoin throttles POST /auth/join per IP — same shape as
// RateLimitLogin, since guessing an organization's shared password is
// exactly a credential-guessing attack, just against a shared secret
// instead of a personal one.
func RateLimitJoin() RateLimitConfig {
	return RateLimitConfig{Name: "join", Limit: 20, Window: 5 * time.Minute, Burst: 6, BurstWindow: 10 * time.Second, KeyFunc: ByIP}
}

// RateLimitJoinByOrg catches distributed guessing of one organization's
// password across many IPs — RateLimitJoin's IP limit alone can't see
// that. Looser than AuthService's own org-password lockout (which
// already locks the org after a handful of failures); this exists so an
// attacker spreading attempts across IPs to dodge RateLimitJoin still
// hits a ceiling tied to the organization they're actually after.
func RateLimitJoinByOrg() RateLimitConfig {
	return RateLimitConfig{Name: "join_org", Limit: 15, Window: 15 * time.Minute, KeyFunc: ByBodyOrgSlug}
}

// RateLimitOrgSettings throttles organization settings changes that
// require re-entering the org password (rename, slug change, password
// rotation, ownership transfer) — same reasoning and shape as
// RateLimitMFAManage: an authenticated session is not sufficient reason
// to allow unlimited password-guessing against the org's own secret.
func RateLimitOrgSettings() RateLimitConfig {
	return RateLimitConfig{Name: "org_settings", Limit: 10, Window: 5 * time.Minute, KeyFunc: ByOrg}
}

func RateLimitRefresh() RateLimitConfig {
	return RateLimitConfig{Name: "refresh", Limit: 60, Window: 5 * time.Minute, Burst: 15, BurstWindow: 10 * time.Second, KeyFunc: ByIP}
}

// RateLimitPasswordReset covers both /forgot-password and
// /reset-password — an attacker probing either one is doing the same
// kind of account-enumeration/abuse.
func RateLimitPasswordReset() RateLimitConfig {
	return RateLimitConfig{Name: "password_reset", Limit: 6, Window: time.Hour, KeyFunc: ByIP}
}

func RateLimitEmailVerification() RateLimitConfig {
	return RateLimitConfig{Name: "email_verification", Limit: 6, Window: time.Hour, KeyFunc: ByIP}
}

// RateLimitAIPerUser and RateLimitAIPerOrg are applied together on AI
// endpoints: the per-user limit stops one seat from monopolizing the
// org's throughput, the per-org limit bounds the org's total spend
// regardless of seat count.
func RateLimitAIPerUser() RateLimitConfig {
	return RateLimitConfig{Name: "ai_user", Limit: 30, Window: time.Hour, Burst: 5, BurstWindow: 10 * time.Second, KeyFunc: ByUser}
}

func RateLimitAIPerOrg() RateLimitConfig {
	return RateLimitConfig{Name: "ai_org", Limit: 200, Window: time.Hour, KeyFunc: ByOrg}
}

func RateLimitUploads() RateLimitConfig {
	return RateLimitConfig{Name: "uploads", Limit: 30, Window: 10 * time.Minute, Burst: 8, BurstWindow: 10 * time.Second, KeyFunc: ByUser}
}

func RateLimitImports() RateLimitConfig {
	return RateLimitConfig{Name: "imports", Limit: 10, Window: time.Hour, KeyFunc: ByUser}
}

func RateLimitEmailsSend() RateLimitConfig {
	return RateLimitConfig{Name: "emails_send", Limit: 100, Window: time.Hour, Burst: 20, BurstWindow: time.Minute, KeyFunc: ByOrg}
}

// RateLimitAPI is the broad safety net applied to the entire protected
// API group — every route gets at least this, even ones without a more
// specific policy above.
func RateLimitAPI() RateLimitConfig {
	return RateLimitConfig{Name: "api", Limit: 600, Window: time.Minute, Burst: 60, BurstWindow: 5 * time.Second, KeyFunc: ByUser}
}

// RateLimitOAuth covers the OAuth start/callback redirects — public,
// unauthenticated, and a plausible target for state-guessing or plain
// flooding since they're reachable without a session.
func RateLimitOAuth() RateLimitConfig {
	return RateLimitConfig{Name: "oauth", Limit: 30, Window: 5 * time.Minute, Burst: 10, BurstWindow: 10 * time.Second, KeyFunc: ByIP}
}

// RateLimitWebhookInbound is for inbound webhook receivers (Notion
// today; Zapier once inbound support exists) — keyed by IP since the
// caller isn't an authenticated user.
func RateLimitWebhookInbound() RateLimitConfig {
	return RateLimitConfig{Name: "webhook_inbound", Limit: 120, Window: time.Minute, Burst: 20, BurstWindow: 5 * time.Second, KeyFunc: ByIP}
}
