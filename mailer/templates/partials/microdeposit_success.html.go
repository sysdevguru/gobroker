package partials

var MicroDepositSuccess Partial = `
{{ define "content" }}
	<img src="https://files.alpaca.markets/webassets/email/success@2x.png" width="128" height="128" style="margin: auto; display: block">
	<div style='text-align:left;'>
		Hello {{ .Name }},<br>
		<br>
		Two small test deposits should be in your bank account ({{ .Nickname }}) at this time. These items will be listed as "Alpaca Securitie ACH".
		<br><br>
		To complete verification of your bank account, enter the amount of the two deposits in your Alpaca account page:<br>
		<a href="https://app.alpaca.markets/banking/microdeposit" style="text-decoration: none; color: #bfa100;">Complete Verification</a>
		<br><br>
		If you have any questions or concerns, please reach out to support@alpaca.markets.<br><br><br>
		Sincerely,<br>
		<br>
		The Alpaca Team<br>
		<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
	</div>
{{ end }}
`
