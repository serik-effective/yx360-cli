package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/calendar"
	"github.com/effective-dev-os/yx360-cli/internal/config"
	"github.com/effective-dev-os/yx360-cli/internal/mail"
	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

// App-password credentials live in their own profiles so they can sit next to
// the OAuth ones: the calendar-telemost OAuth token still serves `telemost
// create`, which has no app-password path.
const (
	mailAppPasswordProfile     = "mail-app-password"
	calendarAppPasswordProfile = "calendar-app-password"
)

func appPasswordProfile(mailScope, mailSendScope, calendarScope, telemostScope, formsScope, diskScope bool) (string, error) {
	switch {
	case telemostScope, formsScope, diskScope:
		return "", errors.New("--app-password covers Mail (IMAP/SMTP) and Calendar (CalDAV) only; Telemost, Forms, and Disk are REST APIs that accept OAuth tokens only, so use yx360 login for those")
	case (mailScope || mailSendScope) && calendarScope:
		return "", errors.New("Yandex issues one app password per service; run yx360 login --app-password --mail and --app-password --calendar separately")
	case mailScope || mailSendScope:
		return mailAppPasswordProfile, nil
	case calendarScope:
		return calendarAppPasswordProfile, nil
	default:
		return "", errors.New("--app-password requires --mail or --calendar")
	}
}

func runAppPasswordLogin(cmd *cobra.Command, profile, account string) error {
	if account == "" {
		account = os.Getenv("YX360_ACCOUNT")
	}
	password, err := readAppPassword(cmd)
	if err != nil {
		return err
	}
	cred, err := auth.NewAppPasswordCredential(account, password)
	if err != nil {
		return err
	}
	if err := verifyAppPassword(cmd.Context(), profile, cred); err != nil {
		return err
	}
	store, err := selectStoreFor(profile)
	if err != nil {
		return err
	}
	if err := store.Save(cmd.Context(), cred); err != nil {
		return err
	}
	payload := loginPayload{Status: "logged-in", Account: cred.Account, Profile: profile, Auth: string(auth.GrantAppPassword)}
	return emit(cmd, humanLogin(payload), payload)
}

func verifyAppPassword(ctx context.Context, profile string, cred *auth.Credential) error {
	if profile == mailAppPasswordProfile {
		return mail.NewService(config.DefaultMail(), cred).Verify(ctx)
	}
	return calendar.NewService(config.DefaultCalendar(), cred).Verify(ctx)
}

// readAppPassword never accepts the password as a flag: flag values leak into
// shell history and `ps` output.
func readAppPassword(cmd *cobra.Command) (string, error) {
	if value := os.Getenv("YX360_APP_PASSWORD"); value != "" {
		return value, nil
	}
	in := cmd.InOrStdin()
	if file, ok := in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprint(cmd.ErrOrStderr(), "App password from "+auth.AppPasswordURL+" (input hidden): ")
		raw, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// loadCredential prefers a stored app password over the OAuth credential of the
// same surface, so `login --app-password` takes effect without a logout.
func loadCredential(ctx context.Context, appPasswordProfile, oauthProfile string) (*auth.Credential, error) {
	store, err := selectStoreFor(appPasswordProfile)
	if err != nil {
		return nil, err
	}
	cred, err := store.Load(ctx)
	if err == nil && cred.IsAppPassword() {
		return cred, nil
	}
	if err != nil && !errors.Is(err, tokenstore.ErrNoCredential) {
		return nil, err
	}
	store, err = selectStoreFor(oauthProfile)
	if err != nil {
		return nil, err
	}
	return store.Load(ctx)
}
