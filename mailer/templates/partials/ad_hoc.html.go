package partials

var AdHoc Partial = `
{{ define "content" }}
	<img src="https://files.alpaca.markets/webassets/email/success@2x.png" width="128" height="128" style="margin: auto; display: block">
	<div style='text-align:left;'>
		{{ .Body }}
	</div>
{{ end }}
`
