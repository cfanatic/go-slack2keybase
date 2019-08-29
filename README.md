# go-slack2keybase

Use this tool to forward chat messages from Slack to Keybase. For those who hate Slack, but love Keybase.

Deploy it on your desktop or server:

Whenever started, your Keybase team will be automatically synchronized with your Slack workspace.

## Requirements

Developed and tested on the following setup:

- macOS 10.13.6
- Go 1.12.7
- Slack 4.0.0
- Keybase 4.2.0

## Installation

You need to set up Slack and Keybase, before you can run the bridge:

1. Log on [**https://api.slack.com**](https://api.slack.com) and create an app for your Slack workspace

2. Under **Features → Bot Users**, define a display and user name for the bot

3. Under **Features → OAuth & Permissions**, select the following permission scopes:

    ```n/a
    Access user’s public channels

    Add a bot user with the username @<your_bot_name>
    ```

4. Go to **Settings → Basic Information** and install the app to your workspace

5. Open **Slack** and add your app into all channels of your workspace

6. Open **Keybase** and create a team which has the same name as the Slack workspace

7. Add channels to your team in **Keybase** according to the ones in your Slack workspace

Remember the **OAuth Access Token** and the **Bot User OAuth Access Token** information.

## Setup

View the package documentation on how to implement the bridge:

`go doc -all github.com/cfanatic/go-slack2keybase/bridge`

### Desktop Mode

Clone the bridge on your desktop:

`go get github.com/cfanatic/go-slack2keybase`

Enter the correct OAuth information in _main.go_:

```Go
const oauth_user = "<INSERT_USER_TOKEN>"
const oauth_bot = "<INSERT_BOT_TOKEN>"
```

Build and install the repository:

`go install github.com/cfanatic/go-slack2keybase`

Run the bridge by calling:

`go-slack2keybase`

Upon successfull installation, you will see output similar to the following screenshot:

![screenshot](https://raw.githubusercontent.com/cfanatic/go-slack2keybase/master/misc/slack2keybase.png)

### Server Mode

Clone the Keybase command line client on your server:

`go get github.com/keybase/client/go/keybase`

Modify [**keybase/client/go/service/main.go**](https://github.com/keybase/client/blob/a648b2fc1b80a3a4c2d5c2b0279cb64669b01bdc/go/service/main.go) to call the bridge before the listen loop:

```Go
// At this point initialization is complete, and we're about to start the
// listen loop. This is the natural point to report "startup successful" to
// the supervisor (currently just systemd on Linux). This isn't necessary
// for correctness, but it allows commands like "systemctl start keybase.service"
// to report startup errors to the terminal, by delaying their return
// until they get this notification (Type=notify, in systemd lingo).
systemd.NotifyStartupFinished()

const oauth_user = "<INSERT_USER_TOKEN>"
const oauth_bot = "<INSERT_BOT_TOKEN>"
bridge := bridge.New(oauth_user, oauth_bot, true)
bridge.Start()
defer bridge.Stop()

d.G().ExitCode, err = d.ListenLoopWithStopper(l)

return err
```

Build and install the repository:

`go install -tags production github.com/keybase/client/go/keybase`

Run the bridge by calling:

`keybase service`

Upon successfull installation, you will see output similar to the following log:

```n/a
server@cfanatic:~/go/bin$ ./keybase service
▶ INFO | net.Listen on unix:/run/user/1000/keybase/keybased.sock
2019/08/27 10:24:57 bridge.go:74: INFO: Slack connection established
2019/08/27 10:24:57 bridge.go:157: INFO: Synchronizing channel "links"
2019/08/27 10:24:58 bridge.go:157: INFO: Synchronizing channel "news"
2019/08/27 10:24:58 bridge.go:157: INFO: Synchronizing channel "education"
2019/08/27 10:24:59 bridge.go:157: INFO: Synchronizing channel "other"
2019/08/27 10:24:59 bridge.go:157: INFO: Synchronizing channel "conferences"
2019/08/27 10:24:59 bridge.go:157: INFO: Synchronizing channel "research"
2019/08/27 10:25:00 bridge.go:157: INFO: Synchronizing channel "general"
2019/08/27 10:25:00 bridge.go:157: INFO: Synchronizing channel "random"
2019/08/27 10:25:02 bridge.go:110: MESSAGE: #random [2019-08-27 10:24:52.0002 +0200 CEST] [Arnd] Test 1-2-3
server@cfanatic:~/go/bin$
```

## Usage

[**This video**](https://codefanatic.de/git/slack2keybase.mp4) demonstrates the general usage and features.
