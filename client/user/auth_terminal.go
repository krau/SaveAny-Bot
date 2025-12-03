package user

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/celestix/gotgproto"
	"golang.org/x/term"
)

type terminalAuthConversator struct{}

func readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func (t *terminalAuthConversator) AskPhoneNumber() (string, error) {
	fmt.Println("Your Phone Number (e.g. +44 123456):")
	return readLine("> ")
}

func (t *terminalAuthConversator) AskCode() (string, error) {
	fmt.Println("Your Code (e.g. 123456):")
	return readLine("> ")
}

func (t *terminalAuthConversator) AskPassword() (string, error) {
	fmt.Println("Your 2FA Password:")
	fmt.Print("> ")
	bytePwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(bytePwd)), nil
}

func (t *terminalAuthConversator) AuthStatus(authStatus gotgproto.AuthStatus) {
	switch authStatus.Event {
	case gotgproto.AuthStatusPhoneRetrial:
		fmt.Printf("The phone number is incorrect. Attempts left: %d\n", authStatus.AttemptsLeft)
	case gotgproto.AuthStatusPasswordRetrial:
		fmt.Printf("The 2FA password is incorrect. Attempts left: %d\n", authStatus.AttemptsLeft)
	case gotgproto.AuthStatusPhoneCodeRetrial:
		fmt.Printf("The OTP code is incorrect. Attempts left: %d\n", authStatus.AttemptsLeft)
	default:
	}
}
