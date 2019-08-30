package partials

var MicroDepositFail Partial = `
{{ define "content" }}
	<img src="https://files.alpaca.markets/webassets/email/failure@2x.png" width="128" height="128" style="margin: auto; display: block">
	<div style='text-align:left;'>
		Hello {{ .Name }},<br>
		<br>
		Two small test deposits to your bank account ({{ .Nickname }}) failed for the following reason.
		<br><br>
		NACHA reason: {{ .Reason }}
		<br><br>
		Please confirm your bank account information and try again from this page:<br>
		<a href="https://app.alpaca.markets/banking/microdeposit" style="text-decoration: none; color: #bfa100;">Confirm Your Bank Account</a>
		<br><br>
		If you have any questions or concerns, please reach out to support@alpaca.markets.<br><br><br>
		Sincerely,<br>
		<br>
		The Alpaca Team<br>
		<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
	</div>
{{ end }}
`
