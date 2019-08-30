package partials

var OrderExecuted Partial = `
{{ define "content" }}
	<div style='text-align:left;'>
		<h3>Your {{ .Symbol }} order has been executed.</h3><br><br>
		Hello {{ .Name }},<br>
		<br>
		Your {{ .OrderType }} order placed for account <a href="https://app.alpaca.markets" style="text-decoration: none; color: #bfa100;">{{ .Account }}</a> has been filled as follows:
		<br>
		<br>
		<table cellspacing="0" cellpadding="0">
			<col align="left">
			<col align="left">
			<col align="right">
			<tr>
				<td>Symbol</td>
				<td style="font-family: Gilroy-ExtraBold; padding:0 0 0 20px;">{{ .Symbol }}</td>
			</tr>
			<tr>
				<td>Side</td>
				<td style="font-family: Gilroy-ExtraBold; padding:0 0 0 20px;">{{ .Side }}</td>
			</tr>
			<tr>
				<td>Quantity</td>
				<td style="font-family: Gilroy-ExtraBold; padding:0 0 0 20px;">{{ .Qty }}</td>
			</tr>
			<tr>
				<td>Price</td>
				<td style="font-family: Gilroy-ExtraBold; padding:0 0 0 20px;">${{ .Price }}</td>
			</tr>
			<tr>
				<td>Timestamp</td>
				<td style="font-family: Gilroy-ExtraBold; padding:0 0 0 20px;">{{ .FilledAt }}</td>
			</tr>
		</table>
		<br>
		Your trade confirmation will be available on your account page Alpaca in one trading day. 
		<br><br>
		If you have any questions or concerns, please visit our <a href="https://support.alpaca.markets" style="text-decoration: none; color: #bfa100;">FAQ page</a>.
		<br><br><br>
		Sincerely,<br>
		<br>
		The Alpaca Team<br>
		<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
	</div>
{{ end }}
`
