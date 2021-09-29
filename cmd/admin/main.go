package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/badoux/checkmail"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"
	"mondaynightpoker-server/pkg/model"
	"os"
	"strings"
)

var command = flag.String("c", "user", "specifies the command (user)")

func main() {
	flag.Parse()

	switch *command {
	case "user":
		email := getEmail()
		if email == "" {
			os.Exit(1)
		}

		password := getPassword()
		if password == "" {
			os.Exit(1)
		}

		player, err := model.CreatePlayer(context.Background(), email, "Admin", password, "127.0.0.1")
		if err != nil {
			logrus.WithError(err).Fatal("could not create player")
		}

		fmt.Printf("Created user %d\n", player.ID)

		name, err := getInput("Name")
		if err != nil {
			logrus.WithError(err).Fatal("could not get answer")
		}
		player.DisplayName = name
		player.Status = model.PlayerStatusVerified
		if err := player.Save(context.Background()); err != nil {
			logrus.WithError(err).Fatal("could not save player")
		}

		promote, err := getInput("Make admin (Y/n)")
		if err != nil {
			logrus.WithError(err).Fatal("could not get answer")
		}

		if promote == "" || strings.ToLower(promote)[0] == 'y' {
			if err := player.SetIsSiteAdmin(context.Background(), true); err != nil {
				logrus.WithError(err).Fatal("could not promote user to admin")
			}

			fmt.Printf("User promoted to admin\n")
		}

	default:
		logrus.Fatalf("unknown command: %s", *command)
	}
}

func getPassword() string {
	for {
		fmt.Print("Password: ")
		pwBytes, err := term.ReadPassword(0)
		if err != nil {
			continue
		}
		fmt.Println("")

		password := strings.TrimRight(string(pwBytes), "\r\n")

		if password == "" {
			return ""
		}

		if len(password) < 6 {
			_, _ = fmt.Fprintf(os.Stderr, "password must be 6 or more characters\n")
			continue
		}

		return password
	}
}

func getEmail() string {
	for {
		fmt.Print("Email: ")
		reader := bufio.NewReader(os.Stdin)
		str, err := reader.ReadString('\n')
		if err != nil {
			logrus.WithError(err).Warn("could not read email")
		}

		str = strings.TrimRight(str, "\r\n")

		if str == "" {
			return ""
		}

		if err := checkmail.ValidateFormat(str); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			continue
		}

		return str
	}
}

func getInput(question string) (string, error) {
	fmt.Printf("%s: ", question)
	reader := bufio.NewReader(os.Stdin)
	str, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	str = strings.TrimRight(str, "\r\n")

	return str, nil
}
