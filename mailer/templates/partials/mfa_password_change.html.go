package partials

var MFAPasswordChange Partial = `
{{ define "content" }}
	<img src="https://files.alpaca.markets/webassets/email/failure@2x.png" width="128" height="128" style="margin: auto; display: block">
	<div style='text-align:left;'>
		Hello {{ .Name }},<br>
		<br>
		Your bank account was unlinked. This may have occurred if you recently changed your password or multi-factor authentication (MFA) settings on your bank account.
		<br><br>
		To re-link your bank account, please log in to your Alpaca account and go to your <a href="https://app.alpaca.markets" style="text-decoration: none; color: #bfa100;">Account Page</a>.<br>
		<br><br>
		If you have any questions or concerns, please reach out to support@alpaca.markets.<br><br><br>
		Sincerely,<br>
		<br>
		The Alpaca Team<br>
		<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
	</div>
{{ end }}
`
