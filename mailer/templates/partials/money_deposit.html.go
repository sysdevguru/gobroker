package partials

var MoneyDeposit Partial = `
{{ define "content" }}
	<div style='text-align:left;'>
		<h3>You have an incoming money transfer.</h3><br><br>
		Hello {{ .Name }},<br>
		<br>
		<a style="font-family: Gilroy-ExtraBold;">${{ .Amount }}</a> was transferred to your account <a href="https://app.alpaca.markets" style="text-decoration: none; color: #bfa100;">{{ .Account }}</a> on {{ .TransferDate }}. The funds will be available for use by the next trading day.
		<br>
		<br>
		If you have any questions or concerns, please visit our <a href="https://support.alpaca.markets" style="text-decoration: none; color: #bfa100;">FAQ page</a>.
		<br><br><br>
		Sincerely,<br>
		<br>
		The Alpaca Team<br>
		<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
	</div>
{{ end }}
`
