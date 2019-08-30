package layouts

import (
	"strings"

	"github.com/alpacahq/gopaca/clock"
)

type Layout string

const year = "{current_year}"

// Base returns email template with current year
func Base() Layout {
	return Layout(strings.Replace(string(base), year, clock.Now().Format("2006"), 1))
}

var base Layout = `
{{ define "layout" }}
<!DOCTYPE html>
<html><head>
  <style type="text/css">
    #PreviewIntercomModal .ic_message_content h1,
    #PreviewIntercomModal .ic_message_content h2 {
      color: #0072b0 !important;
    }

    #PreviewIntercomModal .ic_message_without_image>.ic_message_internals>.ic_message_content {
      border-color: #0072b0 !important;
    }

    #PreviewIntercomModal .ic_user_comment_body {
      background-color: #0072b0 !important;
      border-color: #0072b0 !important;
    }

    #PreviewIntercomModal .ic_message_content a {
      color: #0072b0 !important;
    }

    #PreviewIntercomModal .ic_message_content a:hover {
      color: #0072b0 !important;
    }

    #PreviewIntercomModal .ic_user_comment_body {
      background-color: #0072b0 !important;
      border-color: #0072b0 !important;
    }

    .intercom-h2b-button br {
      display: none;
    }
  </style>
  <style type="text/css" data-premailer="ignore">
    /* styles in here will not be inlined. Use for media queries etc */
    /* force Outlook to provide a "view in browser" menu link. */
    #outlook a {
      padding: 0;
    }

    /* prevent Webkit and Windows Mobile platforms from changing default font sizes.*/
    body {
      width: 100% !important;
      -webkit-text-size-adjust: 100%;
      -ms-text-size-adjust: 100%;
      margin: 0;
      padding: 0;
    }

    /* force Hotmail to display emails at full width */
    .ExternalClass {
      width: 100%;
    }

    /* force Hotmail to display normal line spacing. http://www.emailonacid.com/forum/viewthread/43/ */
    .ExternalClass,
    .ExternalClass p,
    .ExternalClass span,
    .ExternalClass font,
    .ExternalClass td,
    .ExternalClass div {
      line-height: 100%;
    }

    /* fix a padding issue on Outlook 07, 10 */
    table td {
      border-collapse: collapse;
    }

    table {
      table-layout: fixed;
    }

    @media only screen and (max-width: 480px) {
      br.hidden {
        display: block !important;
      }

      td.padding_cell {
        display: none !important;
      }

      table.message_footer_table td {
        font-size: 11px !important;
      }
    }

    @media only screen and (max-device-width: 480px) {
      br.hidden {
        display: block !important;
      }

      td.padding_cell {
        display: none !important;
      }

      table.message_footer_table td {
        font-size: 11px !important;
      }
    }
  </style>

  <style type="text/css" data-premailer="ignore">
    /* styles in here will not be inlined. Use for media queries etc */
    /* force Outlook to provide a "view in browser" menu link. */
    #outlook a {
      padding: 0;
    }

    /* prevent Webkit and Windows Mobile platforms from changing default font sizes.*/
    body {
      width: 100% !important;
      -webkit-text-size-adjust: 100%;
      -ms-text-size-adjust: 100%;
      margin: 0;
      padding: 0;
    }

    /* force Hotmail to display emails at full width */
    .ExternalClass {
      width: 100%;
    }

    /* force Hotmail to display normal line spacing. http://www.emailonacid.com/forum/viewthread/43/ */
    .ExternalClass,
    .ExternalClass p,
    .ExternalClass span,
    .ExternalClass font,
    .ExternalClass td,
    .ExternalClass div {
      line-height: 100%;
    }

    /* fix a padding issue on Outlook 07, 10 */
    table td {
      border-collapse: collapse;
    }

    @media only screen and (max-width: 480px) {
      br.hidden {
        display: block !important;
      }

      td.padding_cell {
        display: none !important;
      }

      table.message_footer_table td {
        font-size: 11px !important;
      }
    }

    @media only screen and (max-device-width: 480px) {
      br.hidden {
        display: block !important;
      }

      td.padding_cell {
        display: none !important;
      }

      table.message_footer_table td {
        font-size: 11px !important;
      }
    }
  </style>

  <style type="text/css">
    .admin_name b {
      color: #6f6f6f;
    }

    .date_cell a {
      color: #999999;
    }

    .comment_header_td {
      width: 100%;
      background: #FCD503;
      border: none;
      font-family: 'Helvetica Neue', Arial, sans-serif
    }

    .content-td {
      color: #525252;
      box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.05);
      font-family: Helvetica, Arial, sans-serif;
    }

    .content-td h1 {
      font-size: 26px;
      line-height: 33px;
      color: #282F33;
      margin-bottom: 7px;
      margin-top: 30px;
      font-weight: normal;
    }

    .content-td h1 a {
      color: #282F33;
    }

    .content-td h2 {
      font-size: 18px;
      font-weight: bold;
      color: #282F33;
      margin: 30px 0 7px;
    }

    .content-td h2 a {
      color: #282F33;
    }

    .content-td h1+h2 {
      margin-top: 0 !important;
    }

    .content-td h2+h1 {
      margin-top: 0 !important;
    }

    .content-td h3,
    .content-td h4,
    .content-td h5 {
      font-size: 16px;
      font-weight: bold;
      margin-bottom: 5px;
    }

    .content-td p {
      margin: 0 0 17px 0;
      line-height: 1.5;
    }

    .content-td p img,
    .content-td h1 img,
    .content-td h2 img,
    .content-td li img,
    .content-td .intercom-h2b-button img {
      margin: 0;
      padding: 0;
    }

    .content-td a {
        color: #BA9527;
        text-decoration: underline;
        font-weight: bold;
    }

    .content-td a:hover {
        color: #fcd503;
    }

    .content-td a:active {
        color: #fcd503;
    }

    .content-td a:visited {
        color: #947120;
    }

    .content-td p.intro {
      font-size: 20px;
      line-height: 30px;
    }

    .content-td blockquote {
      margin: 40px 0;
      font-style: italic;
      color: #8C8C8C;
      font-size: 18px;
      text-align: center;
      padding: 0 30px;
      font-family: Georgia, sans-serif;
      quotes: none;
    }

    .content-td blockquote a {
      color: #8C8C8C;
    }

    .content-td ul {
      list-style: disc;
      margin: 0 0 20px 40px;
      padding: 0;
    }

    .content-td ol {
      list-style: decimal;
      margin: 0 0 20px 40px;
      padding: 0;
    }

    .content-td img {
      margin: 17px 0;
      max-width: 100%;
    }

    .content-td .intercom-container {
      margin-bottom: 16px;
    }

    .content-td hr {
      border: none;
      border-top: 1px solid #DDD;
      border-bottom: 0;
      margin: 50px 30% 50px 30%;
    }

    /**/
    .content-td pre {
      margin: 0 0 10px;
      padding: 10px;
      background-color: #f5f5f5;
      overflow: auto;
    }

    .content-td pre code {
      font-family: Courier, monospace;
      font-size: 14px;
      line-height: 1.4;
      white-space: nowrap;
    }

    table.intercom-container {
      margin: 17px 0;
    }

    table.intercom-container.intercom-align-center {
      margin-left: auto;
      margin-right: auto;
    }

    table.intercom-container td {
      background-color: #FCD503;
      padding: 12px 35px;
      border-radius: 3px;
      font-family: Helvetica, Arial, sans-serif;
      margin: 0;
    }

    .content-td .intercom-h2b-button {
      font-size: 14px;
      color: #2f2f2f !important;
      font-weight: bold;
      display: inline-block;
      text-decoration: none;
      background-color: #fcd503;
      border: none !important;
      margin: 10px 0 10px 0;
      padding: 13px 35px;

    }

    a.intercom-h2b-button {
      background-color: #fcd503;
      border-radius: 5px;
      border: 1px solid rgba(0, 0, 0, 0.2);
      color: #2f2f2f !important;
      display: inline-block;
      font-size: 15px;
      font-weight: bold;
      min-height: 20px;
      text-decoration: none;
    }

    .content-td .intercom-h2b-button:hover {
      background-color: #ffde2b;
    }

    .message_footer_table .avatar {
      -ms-interpolation-mode: bicubic;
      -webkit-background-clip: padding-box;
      -webkit-border-radius: 20px;
      background-clip: padding-box;
      border-radius: 20px;
      display: inline-block;
      height: 40px;
      max-width: 100%;
      outline: none;
      text-decoration: none;
      width: 40px;
    }

    .powered-by-table .powered-by-text a {
      font-weight: bold;
      text-decoration: none;
      color: #999;
    }

    .main_wrapper {
      padding: 0 20px;
    }

    .content-td> :first-child {
      margin-top: 0;
      padding-top: 0;
    }
  </style>

  <!-- Responsive-->
  <style type="text/css" data-premailer="ignore">
    @media screen and (max-width: 635px) {
      .main-wrap {
        width: 100% !important;
      }
    }

    @media screen and (max-width: 480px) {
      .content-td {
        padding: 30px 15px !important;
      }

      .content-td h1 {
        margin-bottom: 5px;
      }

      .message_footer_table .space {
        width: 20px !important;
      }

      .message_footer_table .arrow-wrap {
        padding-left: 20px !important;
      }

      .message_footer_table .admin_name b {
        display: block !important;
      }

      .main_wrapper {
        padding: 0;
      }

      .comment_body {
        border-bottom: 1px solid #DDD !important;
      }

      .footer-td-wrapper {
        display: block !important;
        width: 100% !important;
        text-align: left !important;
      }

      .footer-td-wrapper .date_cell {
        text-align: left !important;
        padding: 15px 0 0 20px !important;
      }
    }
  </style>


  <style type="text/css" data-premailer="ignore">
    .content-td blockquote+* {
      margin-top: 20px !important;
    }

    .ExternalClass .content-td h1 {
      padding: 20px 0 !important;
    }

    .ExternalClass .content-td h2 {
      padding: 0 0 5px !important;
    }

    .ExternalClass .content-td p {
      padding: 10px 0 !important;
    }

    .ExternalClass .content-td .intercom-container {
      padding: 5px 0 !important;
    }

    .ExternalClass .content-td hr+* {
      padding-top: 30px !important;
    }

    .ExternalClass .content-td ol,
    .ExternalClass .content-td ul {
      padding: 0 0 20px 40px !important;
      margin: 0 !important;
    }

    .ExternalClass .content-td ol li,
    .ExternalClass .content-td ul li {
      padding: 3px 0 !important;
      margin: 0 !important;
    }

    .content-td> :first-child {
      margin-top: 0 !important;
      padding-top: 0 !important;
    }

    .ExternalClass .content-td> :first-child {
      margin-top: 0 !important;
      padding-top: 0 !important;
    }
  </style>



  <style type="text/css">
    .intercom-align-right {
      text-align: right !important;
    }

    .intercom-align-center {
      text-align: center !important;
    }

    .intercom-align-left {
      text-align: left !important;
    }

    /* Over-ride for RTL */
    .right-to-left .intercom-align-right {
      text-align: left !important;
    }

    .right-to-left .intercom-align-left {
      text-align: right !important;
    }

    .right-to-left .intercom-align-left {
      text-align: right !important;
    }

    .right-to-left li {
      text-align: right !important;
      direction: rtl;
    }

    .right-to-left .intercom-align-left img,
    .right-to-left .intercom-align-left .intercom-h2b-button {
      margin-left: 0 !important;
    }

    .intercom-attachment,
    .intercom-attachments,
    .intercom-attachments td,
    .intercom-attachments th,
    .intercom-attachments tr,
    .intercom-attachments tbody,
    .intercom-attachments .icon,
    .intercom-attachments .icon img {
      border: none !important;
      box-shadow: none !important;
      padding: 0 !important;
      margin: 0 !important;
    }

    .intercom-attachments {
      margin: 10px 0 !important;
    }

    .intercom-attachments .icon,
    .intercom-attachments .icon img {
      width: 16px !important;
      height: 16px !important;
    }

    .intercom-attachments .icon {
      padding-right: 5px !important;
    }

    .intercom-attachment {
      display: inline-block !important;
      margin-bottom: 5px !important;
    }

    .intercom-interblocks-content-card {
      width: 334px !important;
      max-height: 136px !important;
      max-width: 100% !important;
      overflow: hidden !important;
      border-radius: 20px !important;
      font-size: 16px !important;
      border: 1px solid #e0e0e0 !important;
    }

    .intercom-interblocks-link,
    .intercom-interblocks-article-card {
      text-decoration: none !important;
    }

    .intercom-interblocks-article-icon {
      width: 22.5% !important;
      height: 136px !important;
      float: left !important;
      background-color: #fafafa !important;
      background-image: url('https://alpaca-e339b1d6388e.intercom-mail.com/assets/article_book-1a595be287f73c0d02f548f513bfc831.png') !important;
      background-repeat: no-repeat !important;
      background-size: 32px !important;
      background-position: center !important;
    }

    .intercom-interblocks-article-text {
      width: 77.5% !important;
      float: right !important;
      background-color: #fff !important;
    }

    .intercom-interblocks-link-title,
    .intercom-interblocks-article-title {
      color: #519dd4 !important;
      font-size: 15px !important;
      margin: 16px 18px 12px !important;
      line-height: 1.3em !important;
      overflow: hidden !important;
    }

    .intercom-interblocks-link-description,
    .intercom-interblocks-article-body {
      margin: 0 18px 12px !important;
      font-size: 14px !important;
      color: #65757c !important;
      line-height: 1.3em !important;
    }

    .intercom-interblocks-link-author,
    .intercom-interblocks-article-author {
      margin: 10px 15px !important;
      height: 24px !important;
      line-height: normal !important;
    }

    .intercom-interblocks-link-author-avatar,
    .intercom-interblocks-article-author-avatar {
      width: 16px !important;
      height: 16px !important;
      display: inline-block !important;
      vertical-align: middle !important;
      float: left;
      margin-right: 5px;
    }

    .intercom-interblocks-link-author-avatar-image,
    .intercom-interblocks-article-author-avatar-image {
      width: 16px !important;
      height: 16px !important;
      border-radius: 50% !important;
      margin: 0 !important;
      vertical-align: top !important;
      font-size: 12px !important;
    }

    .intercom-interblocks-link-author-name,
    .intercom-interblocks-article-author-name {
      color: #74848b !important;
      margin: 0 0 0 5px !important;
      font-size: 12px !important;
      font-weight: 500 !important;
      overflow: hidden !important;
    }

    .intercom-interblocks-article-written-by {
      color: #8897a4 !important;
      margin: 1px 0 0 5px !important;
      font-size: 12px !important;
      overflow: hidden !important;
      vertical-align: middle !important;
      float: left !important;
    }
  </style>

  <!-- Alpaca styles and fixes -->
  <style>
    .intercom-container img {
        max-width: 100%;
    }
    .intercom-h2b-button:hover {
        background-color:#ebc703;
    }

    .header-logo {
      width: 140px;
    }
    .container.header {
        margin-bottom: 10px;
    }
    
    .container.content a {
        color: #BA9527;
        text-decoration: underline;
        font-weight: bold;
    }

    .container.footer {
        color: #a1a1a1;
        padding: 25px;
    }
    .container.footer a, .unsubscribe a {
        font-weight: bold;
        text-decoration: underline;
        color: #a1a1a1;
    }
    .container.footer a:hover, container.contenxt a:hover, .unsubscribe a:hover {
        color: #fcd503;
    }

    .container.social-icons {
        padding: 25px 0 0 0;
    }
    table.social-icons th {
        padding-right: 20px;
    }

    td.pad-right {
        padding-right: 20px;
    }
    td.pad-left {
        padding-left: 20px;
    }
    
    .bullet {
        color: #947120;
        font-size: 20px;
        font-weight: bold;
        padding-right: 20px;
    }
    .brown-list {
        color: #947120;
        margin-left: -25px;
    }
    .brown-list p {
        color: #2f2f2f;
    }
    a {
    color: #BA9527;
    text-decoration: underline;
    font-weight: bold;
    }

    a:hover {
    color: #fcd503;
    }

    a:active {
    color: #fcd503;
    }

    a:visited {
    color: #947120;
    }

    .text-left {
    text-align: left;
    }

    .text-center {
    text-align: center;
    }

    .text-right {
    text-align: right;
    }
    .color-brown {
    color: #947120;
    }
    .color-light-brown {
    color: #BA9527;
    }
    .color-yellow, .color-primary {
    color: #FCD503;
    }
    .weight-normal {
        font-weight: normal;
    }
    .float-left {
        float: left;
    }
    .align-left {
        text-align: left;
    }
    .float-right {
        float: right;
    }
    .align-right {
        text-align: right;
    }
    .clear {
        clear: both;
        display: block;
    }
  </style>


</head>

<body>

  <table cellpadding="0" cellspacing="0" border="0" class="bgtc personal" align="center" style="border-collapse: collapse; line-height: 100% !important; margin: 0;   padding: 0; width: 100% !important; background-color: #fff7cc;">
    <tbody>
      <tr>
        <td>
          <!--[if (gte mso 10)]>
      <tr>
      <td>
      <table style="width: 600px">
    <![endif]-->
          <table style="border-collapse: collapse; margin: auto; width: 100%; max-width: 635px; min-width: 320px;  " class="main-wrap">
            <tbody>
              <tr>
                <td valign="top">
                  <table cellpadding="0" cellspacing="0" border="0" class="reply_header_table" style="border-collapse: collapse; color: #c0c0c0; font-family: 'Helvetica Neue',Arial,sans-serif; font-size: 13px; line-height: 26px; margin: 0 auto 26px;   width: 100%">
                  </table>
                </td>
              </tr>
              <tr>
                <td valign="top" class="main_wrapper">

                <div class="container header">
                    <img class="header-logo" src="https://s3.amazonaws.com/marketing.alpaca.markets/emails/Alpaca_Logo_Header.png" alt="Alpaca">
                </div>

                  <table cellpadding="0" cellspacing="0" border="0" class="comment_wrapper_table admin_comment" align="center" style="-webkit-background-clip: padding-box; -webkit-border-radius: 3px; background-clip: padding-box; border-collapse: collapse; border-radius: 3px; color: #545454; font-family: 'Helvetica Neue',Arial,sans-serif; font-size: 13px; line-height: 20px; margin: 0 auto;   width: 100%">
                    <tbody>
                      <tr>
                        <td valign="top" class="comment_wrapper_td">
                          <table cellpadding="0" cellspacing="0" border="0" class="comment_header" style="border-collapse: separate; border: none; font-size: 1px; height: 2px; line-height: 3px;   width: 100%">
                            <tbody>
                              <tr>
                                <td valign="top" class="comment_header_td"></td>
                              </tr>
                            </tbody>
                          </table>
                          <table cellpadding="0" cellspacing="0" border="0" class="comment_body" style="-webkit-background-clip: padding-box; -webkit-border-radius: 0 0 3px 3px; background-clip: padding-box; border-collapse: collapse; border-color: #dddddd; border-radius: 0 0 3px 3px; border-style: solid; border-width: 0 1px 1px;   width: 100%; border-bottom: none">
                            <tbody>
                              <tr>
                                <td class="comment_body_td content-td" style="-webkit-background-clip: padding-box; -webkit-border-radius: 0 0 3px 3px; background-clip: padding-box; border-radius: 0 0 3px 3px; color: #525252; font-family: 'Helvetica Neue',Arial,sans-serif; font-size: 15px; line-height: 22px; overflow: hidden; padding: 40px 40px 30px; background-color: white;">



                                {{ template "content" . }}



                                </td>
                              </tr>
                            </tbody>
                          </table>
                        </td>
                      </tr>
                    </tbody>
                  </table>

                  <table width="100%" cellpadding="0" cellspacing="0" border="0">
                    <tbody>
                      <tr>
                        <td width="100%" class="footer-td-wrapper">
                          <table width="100%" cellpadding="0" cellspacing="0" border="0" style="border-collapse: collapse; color: #545454; font-family: 'Helvetica Neue',Arial,sans-serif; font-size: 13px; line-height: 20px; margin: 0 auto; max-width: 100%;   width: 100%" class="message_footer_table">
                            <tbody>
                              <tr>
                                <td valign="middle" align="right" class="date_cell" style="color: #999999; text-align: center; font-size: 12px;">

                                    <div class="container social-icons">
                                        <table align="center" class="menu social-icons float-center">
                                            <tbody>
                                            <tr>
                                                <td>
                                                <table>
                                                    <tbody><tr>
                                                    <!-- change all social icons by modifying the src and make sure to add link destinations by replacing href="#"-->
                                                    <th class="menu-item float-center"><a href="https://twitter.com/alpacahq" target="_blank"><img src="https://s3.amazonaws.com/marketing.alpaca.markets/emails/twitter-gray.png" alt="Twitter" class="social-icon" height="20"></a></th>
                                                    <th class="menu-item float-center"><a href="https://www.facebook.com/alpacahq" target="_blank"><img src="https://s3.amazonaws.com/marketing.alpaca.markets/emails/facebook-gray.png" alt="Facebook" class="social-icon" height="20"></a></th>
                                                    <th class="menu-item float-center"><a href="https://www.linkedin.com/company/9300556/" target="_blank"><img src="https://s3.amazonaws.com/marketing.alpaca.markets/emails/linkedin-gray.png" alt="LinkedIn" class="social-icon" height="20"></a></th>
                                                    <th class="menu-item float-center"><a href="https://github.com/alpacahq" target="_blank"><img src="https://s3.amazonaws.com/marketing.alpaca.markets/emails/github-gray.png" alt="GitHub" class="social-icon" height="20"></a></th>
                                                    </tr>
                                                </tbody></table>
                                                </td>
                                            </tr>
                                            </tbody>
                                        </table>
                                    </div>
                                        
                                    <div class="container footer">
                                        <p class="text-left">
                                            <small>
                                                Alpaca Securities LLC ("Alpaca") is a wholly-owned subsidiary of AlpacaDB, Inc. Brokerage services are provided to self-directed customers by Alpaca, member FINRA/SIPC. Free trading refers to $0 commissions for Alpaca self-directed individual cash or margin brokerage accounts that trade U.S. listed securities via the Application Programming Interface. Relevant SEC & FINRA fees may apply. This is not an offer, solicitation of an offer, or advice to buy or sell securities, or open a brokerage account in any jurisdiction where Alpaca is not registered.
                                                <br><br>
                                                Alpaca Securities LLC does not take responsibility for, or accept time-sensitive instructions sent by email including orders or funds transfer instructions.
                                                <br><br>
                                                The Paper Trading API is offered by AlpacaDB, Inc. and does not require real money or permit a user to transact in real securities in the market.  Providing use of the Paper Trading API is not an offer or solicitation to buy or sell securities, securities derivative or futures products of any kind, or any type of trading or investment advice, recommendation or strategy, given or in any manner endorsed by AlpacaDB, Inc. or any AlpacaDB, Inc. affiliate and the information made available through the Paper Trading API is not an offer or solicitation of any kind in any jurisdiction where AlpacaDB, Inc. or any AlpacaDB, Inc. affiliate is not authorized to do business.
                                                <br><br>
                                                The information transmitted is intended only for the person or entity to which it is addressed and may contain confidential and/or privileged material. Any review, retransmission, dissemination or other use of, or taking of any action in reliance upon, this information by persons or entities other than the intended recipient is prohibited. If you have received this communication in error, please destroy all electronic and paper copies and notify the sender immediately. This message cannot be guaranteed to be secure or error-free. Alpaca reserves the right, to the extent required and/or permitted under applicable law, to monitor electronic communications.
                                            </small>
                                            </p>
                                            <br>
                                    
                                            <p class="text-center copyright">
                                            <small>
                                                © {current_year} Alpaca Securities LLC. All rights reserved.
                                            </small>
                                            </p>
                                            <p class="text-center">
                                            <small>
                                                <a href="https://maps.google.com/?q=217+B+St+%232,+San+Mateo,+CA+94401&amp;entry=gmail&amp;source=g" target="_blank">
                                                217 B St. #2, San Mateo, CA 94401
                                                </a>
                                            </small>
                                            </p>
                                            <p class="text-center unsubscribe">
                                            <small>
                                            </small>
                                            </p>
                                        </div>
                                </td>
                              </tr>
                              <tr>
                              </tr>
                            </tbody>
                          </table>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </td>
              </tr>
            </tbody>
          </table>
        </td>
      </tr>
      <tr>
        <td valign="top" height="20"></td>
      </tr>
    </tbody>
  </table>
  <!--[if (gte mso 10)]>
            </td>
            </tr>
            </table>
          <![endif]-->
</body></html>
{{ end }}
`
