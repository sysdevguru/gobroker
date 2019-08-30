package partials

var BalanceLow Partial = `
{{ define "content" }}
	<img src="https://files.alpaca.markets/webassets/email/failure@2x.png" width="128" height="128" style="margin: auto; display: block">
	<div style='text-align:left;'>
		Hello {{ .Name }},<br>
		<br>
		Your bank returned an insufficient funds message. Please contact your bank for additional information.
		<br><br>
		Once your account balance has been confirmed, please try resubmitting your incoming transfer on your <a href="https://app.alpaca.markets" style="text-decoration: none; color: #bfa100;">Account Page</a>.<br>
		<br><br>
		If you have any questions or concerns, please reach out to support@alpaca.markets.<br><br><br>
		Sincerely,<br>
		<br>
		The Alpaca Team<br>
		<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
	</div>
{{ end }}
`
