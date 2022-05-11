package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	google_oauth2 "golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"

	"github.com/robbydyer/sports/internal/gcal"
)

const gcaldesc = `
Before running this command, perform the following steps:

- You will first have to create a project at cloud.google.com
- Enable the Google Calendar API at https://console.cloud.google.com/apis/dashboard.
- Follow the steps for creating Oauth client ID credentials for a "Desktop app" here:
https://developers.google.com/workspace/guides/create-credentials#oauth-client-id. You will
download the JSON credentials, and save them to the Pi as a file named /etc/google_calendar_credentials.json.
`

type gcalSetupCmd struct {
	rArgs *rootArgs
}

func newGcalSetupCmd(args *rootArgs) *cobra.Command {
	g := gcalSetupCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "gcalsetup",
		Short: "Performs initial authentication setup for Google Calendar",
		Long:  gcaldesc,
		RunE:  g.run,
	}

	return cmd
}

func (g *gcalSetupCmd) run(cmd *cobra.Command, args []string) error {
	b, err := ioutil.ReadFile(gcal.CredentialsFile)
	if err != nil {
		return err
	}

	config, err := google_oauth2.ConfigFromJSON(b, calendar.CalendarEventsReadonlyScope, calendar.CalendarReadonlyScope)
	if err != nil {
		return err
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return err
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(gcal.TokenFile, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(tok)
}
