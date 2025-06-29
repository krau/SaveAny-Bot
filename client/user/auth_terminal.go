package user

import (
	"strings"

	"github.com/celestix/gotgproto"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/fatih/color"
)

type terminalAuthConversator struct{}

func (t *terminalAuthConversator) AskPhoneNumber() (string, error) {
	phone := ""
	err := huh.NewInput().Title("Your Phone Number").
		Placeholder("+44 123456").
		Prompt("> ").
		Value(&phone).
		WithTheme(huh.ThemeCatppuccin()).
		Run()

	if err != nil {
		return "", err
	}

	log.Info("Sending code to your phone number...")

	return strings.TrimSpace(phone), nil
}

func (t *terminalAuthConversator) AskCode() (string, error) {
	code := ""
	err := huh.NewInput().Title("Your Code").
		Placeholder("123456").
		Value(&code).
		Prompt("> ").
		WithTheme(huh.ThemeCatppuccin()).
		Run()

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(code), nil
}

func (t *terminalAuthConversator) AskPassword() (string, error) {
	pwd := ""

	err := huh.NewInput().Title("Your 2FA Password").
		EchoMode(huh.EchoModePassword).
		Value(&pwd).
		Prompt("> ").
		WithTheme(huh.ThemeCatppuccin()).
		Run()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(pwd), nil
}

func (t *terminalAuthConversator) AuthStatus(authStatus gotgproto.AuthStatus) {
	switch authStatus.Event {
	case gotgproto.AuthStatusPhoneRetrial:
		color.Red("The phone number you just entered seems to be incorrect,")
		color.Red("Attempts Left: %d", authStatus.AttemptsLeft)
		color.Red("Please try again....")
	case gotgproto.AuthStatusPasswordRetrial:
		color.Red("The 2FA password you just entered seems to be incorrect,")
		color.Red("Attempts Left: %d", authStatus.AttemptsLeft)
		color.Red("Please try again....")
	case gotgproto.AuthStatusPhoneCodeRetrial:
		color.Red("The OTP you just entered seems to be incorrect,")
		color.Red("Attempts Left: %d", authStatus.AttemptsLeft)
		color.Red("Please try again....")
	default:
	}
}
