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

type Forms struct {
	BaseURL    string
	OrgID      string
	ReadScope  string
	WriteScope string
}

type Disk struct {
	BaseURL    string
	ReadScope  string
	WriteScope string
}

const defaultClientID = ""
const MailReadScope = "mail:imap_full"
const MailSendScope = "mail:smtp"
const CalendarScope = "calendar:all"
const TelemostCreateScope = "telemost-api:conferences.create"
const FormsReadScope = "forms:read"
const FormsWriteScope = "forms:write"
const DiskReadScope = "cloud_api:disk.read"
const DiskWriteScope = "cloud_api:disk.write"
const VerificationCodeRedirectURI = "https://oauth.yandex.ru/verification_code"

func CalendarClientID() string {
	return os.Getenv("YX360_CALENDAR_CLIENT_ID")
}

func FormsClientID() string {
	return os.Getenv("YX360_FORMS_CLIENT_ID")
}

func FormsOrgID() string {
	return os.Getenv("YX360_FORMS_ORG_ID")
}

func DiskClientID() string {
	return os.Getenv("YX360_DISK_CLIENT_ID")
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

func DefaultForms() Forms {
	baseURL := os.Getenv("YX360_FORMS_API_URL")
	if baseURL == "" {
		baseURL = "https://api.forms.yandex.net"
	}
	return Forms{BaseURL: baseURL, OrgID: FormsOrgID(), ReadScope: FormsReadScope, WriteScope: FormsWriteScope}
}

func DefaultDisk() Disk {
	baseURL := os.Getenv("YX360_DISK_API_URL")
	if baseURL == "" {
		baseURL = "https://cloud-api.yandex.net/v1/disk"
	}
	return Disk{BaseURL: baseURL, ReadScope: DiskReadScope, WriteScope: DiskWriteScope}
}
