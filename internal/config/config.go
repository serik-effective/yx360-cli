package config

import "os"

type OAuth struct {
	ClientID      string
	AuthURL       string
	TokenURL      string
	DeviceAuthURL string
	RedirectURI   string
	LoopbackPort  int
	Scopes        []string
}

type Mail struct {
	IMAPHost  string
	IMAPPort  int
	SMTPHost  string
	SMTPPort  int
	ReadScope string
	SendScope string
}

type Calendar struct {
	BaseURL string
	Scope   string
}

type Telemost struct {
	BaseURL     string
	CreateScope string
}

const defaultClientID = ""
const MailReadScope = "mail:imap_full"
const MailSendScope = "mail:smtp"
const CalendarScope = "calendar:all"
const TelemostCreateScope = "telemost-api:conferences.create"

func CalendarClientID() string {
	return os.Getenv("YX360_CALENDAR_CLIENT_ID")
}

func Default() OAuth {
	clientID := os.Getenv("YX360_CLIENT_ID")
	if clientID == "" {
		clientID = defaultClientID
	}
	return OAuth{
		ClientID:      clientID,
		AuthURL:       "https://oauth.yandex.ru/authorize",
		TokenURL:      "https://oauth.yandex.ru/token",
		DeviceAuthURL: "https://oauth.yandex.ru/device/code",
		RedirectURI:   "http://localhost:8899",
		LoopbackPort:  8899,
		Scopes:        []string{"login:info"},
	}
}

func DefaultMail() Mail {
	imapHost := os.Getenv("YX360_IMAP_HOST")
	if imapHost == "" {
		imapHost = "imap.yandex.ru"
	}
	smtpHost := os.Getenv("YX360_SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.yandex.ru"
	}
	return Mail{
		IMAPHost:  imapHost,
		IMAPPort:  993,
		SMTPHost:  smtpHost,
		SMTPPort:  465,
		ReadScope: MailReadScope,
		SendScope: MailSendScope,
	}
}

func DefaultCalendar() Calendar {
	baseURL := os.Getenv("YX360_CALDAV_URL")
	if baseURL == "" {
		baseURL = "https://caldav.yandex.ru"
	}
	return Calendar{BaseURL: baseURL, Scope: CalendarScope}
}

func DefaultTelemost() Telemost {
	baseURL := os.Getenv("YX360_TELEMOST_API_URL")
	if baseURL == "" {
		baseURL = "https://cloud-api.yandex.net/v1/telemost-api"
	}
	return Telemost{BaseURL: baseURL, CreateScope: TelemostCreateScope}
}
