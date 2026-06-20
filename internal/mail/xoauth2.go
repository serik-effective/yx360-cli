package mail

import "github.com/emersion/go-sasl"

type xoauth2Client struct {
	username string
	token    string
}

func newXOAUTH2Client(username, token string) sasl.Client {
	return &xoauth2Client{username: username, token: token}
}

func (c *xoauth2Client) Start() (string, []byte, error) {
	return "XOAUTH2", []byte("user=" + c.username + "\x01auth=Bearer " + c.token + "\x01\x01"), nil
}

func (c *xoauth2Client) Next(_ []byte) ([]byte, error) {
	return nil, sasl.ErrUnexpectedServerChallenge
}
