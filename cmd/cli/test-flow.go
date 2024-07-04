package main

import (
	"github.com/alecthomas/kingpin"
	oidc2 "github.com/coreos/go-oidc/v3/oidc"
	log "github.com/echocat/slf4g"
	"github.com/engity/pam-oidc/pkg/core"
	"golang.org/x/oauth2"
)

func registerTestFlowCmd(app *kingpin.Application) {
	cmd := app.Command("test-flow", "Used to test the flow on command line without PAM").
		Action(func(*kingpin.ParseContext) error {
			return doTestFlow()
		})
	cmd.Arg("configuration", "Configuration which should be used to test the flow.").
		Required().
		SetValue(&configurationRef)
	cmd.Arg("username", "Username which should be used as requested.").
		Required().
		StringVar(&requestedUsername)
}

func doTestFlow() error {
	cord, err := core.NewCoordinator(configurationRef.Get())
	if err != nil {
		return err
	}

	cord.OnDeviceAuthStarted = func(dar *oauth2.DeviceAuthResponse) error {
		if v := dar.VerificationURIComplete; v != "" {
			log.Infof("Open %s in your browser and approve the login request. Waiting for approval...", v)
		} else {
			log.Infof("Open %s in your browser and enter the code %s. Waiting for approval...", dar.VerificationURI, dar.UserCode)
		}
		return nil
	}

	cord.OnTokenReceived = func(token *oauth2.Token) error {
		log.With("token", token).
			Info("Token received.")
		return nil
	}

	cord.OnUserInfoReceived = func(userInfo *oidc2.UserInfo) error {
		log.With("userInfo", userInfo).
			Info("UserInfo received.")
		return nil
	}

	u, err := cord.Run(nil, requestedUsername)
	if err != nil {
		return err
	}

	log.With("user", u).
		Info("User resolved.")

	return nil
}