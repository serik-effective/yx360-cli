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

const defaultClientID = ""

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
