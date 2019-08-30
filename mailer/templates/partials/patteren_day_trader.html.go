package partials

var PatternDayTrader Partial = `
{{ define "content" }}
	<div style='text-align:left;'>
		Hello {{ .Name }},<br>
		<br>
		Your account ({{ .Account }}) has been issued a day-trading margin call.<br>
		<br>
		<b>Please deposit ${{ .Amount }} by {{ .DueDate }} to avoid suspension of the account.<b><br>
		<br>
		See the FAQ section at https://alpaca.markets, if you have any question or concern
		<br>
		<br>
		Please be advised that this call has been issued due to inadequate equity in your 
		account at the time you either met the definition of a "pattern day trader" or if 
		you had previously met the definition, you placed day trades in your account 
		without meeting the minimum equity requirements or you exceeded your day-trading 
		buying power limitations. For additional information on account requirements in 
		an account designated as a pattern day-trading account, see FINRA Rule 4210 - Margin 
		Requirements and http://www.finra.org/investors/day-trading-margin-requirements-know-rules.
		<br><br><br>
		Sincerely,<br>
		<br>
		The Alpaca Team<br>
		<a href="https://alpaca.markets" style="text-decoration: none; color: #bfa100;">https://alpaca.markets</a>
	</div>
{{ end }}
`
