## Google Calendar Setup
In order to enable the Google Calendar board, you have to perform a fairly convoluted setup to enable
authentication.

- Create a project at cloud.google.com
- Enable the Google Calendar API in your new project at https://console.cloud.google.com/apis/dashboard
- Follow the steps for creating an Oauth2 client ID credentials for a "Desktop App" here and be sure to download the JSON credentials you create: https://developers.google.com/workspace/guides/create-credentials#oauth-client-id
- Place the download JSON credentials file on your Pi named `/etc/google_calendar_credentials.json`
- Run the helper command to generate a token on your Pi, following the prompts: `sudo sportsmatrix gcalsetup`
- If successful, you will now have a file named `/etc/google_calendar_token.json`
