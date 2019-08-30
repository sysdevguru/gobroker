package partials

var AccountApproved Partial = `
{{ define "content" }}
	<img src="https://files.alpaca.markets/webassets/email/success@2x.png" width="128" height="128" style="margin: auto; display: block">
	<div style='text-align:left;'>
		<h3>Congratulations! Your Account Has Been Approved.</h3><br><br>
		Hello {{ .Name }},<br>
		<br>
		Your account <a href="https://app.alpaca.markets" style="text-decoration: none; color: #bfa100;">{{ .Account }}</a> has been approved.
		<br>
		If you have any questions or concerns, please visit our <a href="https://support.alpaca.markets" style="text-decoration: none; color: #bfa100;">FAQ page</a>.				<br><br><br>
		Sincerely,<br>
		<br>
		The Alpaca Team<br>
		<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
	</div>
{{ end }}
`
