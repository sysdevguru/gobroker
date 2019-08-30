package partials

var MarginCall Partial = `
{{ define "content" }}
	Hello {{ .Name }},<br>
	<br>
	Your account ({{ .Account }}) has been issued a margin call.<br>
	<br>
	<b>Please deposit ${{ .Amount }} by {{ .DueDate }} to avoid a sell out to cover the call.</b><br>
	<br>
	See the FAQ section at https://alpaca.markets, if you have any question or concern.
	Please be advised that this is a courtesy message and based on market conditions, 
	risk factors, and/or other criteria, we can sell any of your securities used as 
	collateral at our discretion without contacting you to cover this call.
	<br><br><br>
	Kind regards,<br>
	<br>
	The Alpaca Team<br>
	<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
{{ end }}
`
