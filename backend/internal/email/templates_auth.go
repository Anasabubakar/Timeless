package email

import "fmt"

// VerificationEmail builds the "confirm your email" message. Kept as a
// plain Go function (not the html/template path used by user-authored
// campaign emails) since the content is fixed and shouldn't be editable —
// nothing user-controlled is interpolated into it besides the link itself.
func VerificationEmail(to, fromAddr, fromName, verifyURL string) *Message {
	text := fmt.Sprintf("Confirm your email address by visiting:\n\n%s\n\nThis link expires soon and can only be used once. If you didn't create a Timeless account, you can ignore this email.", verifyURL)
	html := fmt.Sprintf(`<p>Confirm your email address to finish setting up your Timeless account.</p><p><a href="%s">Verify my email</a></p><p>This link expires soon and can only be used once. If you didn't create a Timeless account, you can ignore this email.</p>`, verifyURL)

	return &Message{
		From:     fromAddr,
		FromName: fromName,
		To:       []string{to},
		Subject:  "Confirm your email address",
		TextBody: text,
		HTMLBody: html,
		Tags:     map[string]string{"category": "auth.verify_email"},
	}
}

// PasswordResetEmail builds the "reset your password" message.
func PasswordResetEmail(to, fromAddr, fromName, resetURL string) *Message {
	text := fmt.Sprintf("We received a request to reset your Timeless password. Reset it here:\n\n%s\n\nThis link expires soon and can only be used once. If you didn't request this, you can safely ignore this email — your password won't change.", resetURL)
	html := fmt.Sprintf(`<p>We received a request to reset your Timeless password.</p><p><a href="%s">Reset my password</a></p><p>This link expires soon and can only be used once. If you didn't request this, you can safely ignore this email — your password won't change.</p>`, resetURL)

	return &Message{
		From:     fromAddr,
		FromName: fromName,
		To:       []string{to},
		Subject:  "Reset your password",
		TextBody: text,
		HTMLBody: html,
		Tags:     map[string]string{"category": "auth.password_reset"},
	}
}

// InvitationEmail builds the "you've been invited" message sent when
// someone invites a new email address to join their organization.
func InvitationEmail(to, orgName, acceptURL, fromAddr, fromName string) *Message {
	text := fmt.Sprintf("You've been invited to join %s on Timeless. Accept your invitation here:\n\n%s\n\nThis link expires in 7 days and can only be used once. If you weren't expecting this, you can ignore this email.", orgName, acceptURL)
	html := fmt.Sprintf(`<p>You've been invited to join <strong>%s</strong> on Timeless.</p><p><a href="%s">Accept invitation</a></p><p>This link expires in 7 days and can only be used once. If you weren't expecting this, you can ignore this email.</p>`, orgName, acceptURL)

	return &Message{
		From:     fromAddr,
		FromName: fromName,
		To:       []string{to},
		Subject:  fmt.Sprintf("You've been invited to join %s on Timeless", orgName),
		TextBody: text,
		HTMLBody: html,
		Tags:     map[string]string{"category": "team.invitation"},
	}
}

// PasswordChangedEmail notifies a user their password changed, so they can
// react quickly if it wasn't them.
func PasswordChangedEmail(to, fromAddr, fromName string) *Message {
	text := "Your Timeless password was just changed. If this was you, no action is needed. If you didn't make this change, reset your password immediately and contact support."
	html := "<p>Your Timeless password was just changed.</p><p>If this was you, no action is needed. If you didn't make this change, reset your password immediately and contact support.</p>"

	return &Message{
		From:     fromAddr,
		FromName: fromName,
		To:       []string{to},
		Subject:  "Your password was changed",
		TextBody: text,
		HTMLBody: html,
		Tags:     map[string]string{"category": "auth.password_changed"},
	}
}
