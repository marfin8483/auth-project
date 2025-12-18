package utils

import (
	"auth-api/config"
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
)

type EmailData struct {
	Name    string
	OTP     string
	Minutes int
}

type PasswordResetEmailData struct {
	Name    string
	OTP     string
	Minutes int
}

func SendOTPEmail(cfg *config.Config, to, name, otp string, minutes int) error {
	emailTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verifikasi OTP Login</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f4f4f4;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
            border-radius: 10px 10px 0 0;
        }
        .content {
            background: white;
            padding: 40px;
            border-radius: 0 0 10px 10px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .otp-box {
            background: #f8f9fa;
            border: 2px dashed #667eea;
            border-radius: 8px;
            padding: 20px;
            margin: 30px 0;
            text-align: center;
        }
        .otp-code {
            font-size: 36px;
            letter-spacing: 10px;
            font-weight: bold;
            color: #667eea;
            margin: 10px 0;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            color: #666;
            font-size: 14px;
        }
        .warning {
            background: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 5px;
            padding: 15px;
            margin: 20px 0;
            color: #856404;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Verifikasi Login</h1>
            <p>Sistem Autentikasi API</p>
        </div>
        <div class="content">
            <h2>Halo {{.Name}},</h2>
            <p>Kami menerima permintaan login ke akun Anda. Gunakan kode OTP berikut untuk menyelesaikan proses login:</p>
            
            <div class="otp-box">
                <p style="margin: 0 0 10px 0; color: #666;">Kode OTP Anda:</p>
                <div class="otp-code">{{.OTP}}</div>
                <p style="margin: 10px 0 0 0; color: #666;">Berlaku selama {{.Minutes}} menit</p>
            </div>
            
            <div class="warning">
                ⚠️ <strong>PERINGATAN KEAMANAN:</strong>
                <ul style="margin: 10px 0 0 0; padding-left: 20px;">
                    <li>Jangan pernah membagikan kode OTP ini kepada siapapun</li>
                    <li>Tim kami tidak akan pernah meminta kode OTP Anda</li>
                    <li>Kode OTP ini hanya untuk satu kali penggunaan</li>
                </ul>
            </div>
            
            <p>Jika Anda tidak melakukan permintaan login ini, harap abaikan email ini atau hubungi administrator sistem.</p>
            
            <div class="footer">
                <p>Email ini dikirim secara otomatis, harap tidak membalas email ini.</p>
                <p>© 2024 Sistem Autentikasi API. All rights reserved.</p>
            </div>
        </div>
    </div>
</body>
</html>`

	// Parse template
	tmpl, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %v", err)
	}

	// Execute template
	var body bytes.Buffer
	data := EmailData{
		Name:    name,
		OTP:     otp,
		Minutes: minutes,
	}

	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %v", err)
	}

	// SMTP Configuration
	from := cfg.SMTP.From
	password := cfg.SMTP.Password
	smtpHost := cfg.SMTP.Host
	smtpPort := cfg.SMTP.Port

	// Authentication
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Email headers
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("Authentication System <%s>", from)
	headers["To"] = to
	headers["Subject"] = fmt.Sprintf("[%s] Kode OTP Verifikasi Login", otp)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// Build message
	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body.String())

	// Send email
	err = smtp.SendMail(
		fmt.Sprintf("%s:%d", smtpHost, smtpPort),
		auth,
		from,
		[]string{to},
		[]byte(msg.String()),
	)

	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

func SendPasswordResetEmail(cfg *config.Config, to, name, otp string, minutes int) error {
	emailTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Password</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f4f4f4;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            color: white;
            padding: 30px;
            text-align: center;
            border-radius: 10px 10px 0 0;
        }
        .content {
            background: white;
            padding: 40px;
            border-radius: 0 0 10px 10px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .otp-box {
            background: #f8f9fa;
            border: 2px dashed #f5576c;
            border-radius: 8px;
            padding: 20px;
            margin: 30px 0;
            text-align: center;
        }
        .otp-code {
            font-size: 36px;
            letter-spacing: 10px;
            font-weight: bold;
            color: #f5576c;
            margin: 10px 0;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            color: #666;
            font-size: 14px;
        }
        .warning {
            background: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 5px;
            padding: 15px;
            margin: 20px 0;
            color: #856404;
        }
        .btn {
            display: inline-block;
            padding: 12px 30px;
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            color: white;
            text-decoration: none;
            border-radius: 5px;
            margin: 10px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Reset Password</h1>
            <p>Sistem Autentikasi API</p>
        </div>
        <div class="content">
            <h2>Halo {{.Name}},</h2>
            <p>Kami menerima permintaan reset password untuk akun Anda. Gunakan kode OTP berikut untuk melanjutkan proses reset password:</p>
            
            <div class="otp-box">
                <p style="margin: 0 0 10px 0; color: #666;">Kode OTP Reset Password:</p>
                <div class="otp-code">{{.OTP}}</div>
                <p style="margin: 10px 0 0 0; color: #666;">Berlaku selama {{.Minutes}} menit</p>
            </div>
            
            <div class="warning">
                ⚠️ <strong>PERINGATAN KEAMANAN:</strong>
                <ul style="margin: 10px 0 0 0; padding-left: 20px;">
                    <li>Jangan pernah membagikan kode OTP ini kepada siapapun</li>
                    <li>Kode ini hanya untuk reset password</li>
                    <li>Jika Anda tidak meminta reset password, abaikan email ini</li>
                </ul>
            </div>
            
            <p>Setelah verifikasi OTP berhasil, Anda dapat mengganti password Anda.</p>
            
            <div class="footer">
                <p>Email ini dikirim secara otomatis, harap tidak membalas email ini.</p>
                <p>© 2024 Sistem Autentikasi API. All rights reserved.</p>
            </div>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("password_reset_email").Parse(emailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %v", err)
	}

	var body bytes.Buffer
	data := PasswordResetEmailData{
		Name:    name,
		OTP:     otp,
		Minutes: minutes,
	}

	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %v", err)
	}

	from := cfg.SMTP.From
	password := cfg.SMTP.Password
	smtpHost := cfg.SMTP.Host
	smtpPort := cfg.SMTP.Port

	auth := smtp.PlainAuth("", from, password, smtpHost)

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("Authentication System <%s>", from)
	headers["To"] = to
	headers["Subject"] = fmt.Sprintf("[%s] Kode OTP Reset Password", otp)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body.String())

	err = smtp.SendMail(
		fmt.Sprintf("%s:%d", smtpHost, smtpPort),
		auth,
		from,
		[]string{to},
		[]byte(msg.String()),
	)

	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
