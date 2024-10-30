package cli

import (
	"encoding/json"
	"fmt"
	"github.com/gaarutyunov/gitstat/gitlab"
	"github.com/gaarutyunov/gitstat/models"
	"github.com/gaarutyunov/gitstat/types"
	"github.com/gaarutyunov/gitstat/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/url"
)

var cmd = &cobra.Command{
	Use: "gitstat",
	RunE: func(cmd *cobra.Command, args []string) error {
		pFlags := cmd.PersistentFlags()

		server, err := pFlags.GetString("server")
		if err != nil {
			return err
		}
		host, err := pFlags.GetString("host")
		if err != nil {
			return err
		}
		token, err := pFlags.GetString("token")
		if err != nil {
			return err
		}
		rateLimit, err := pFlags.GetInt("rate")
		if err != nil {
			return err
		}
		userAliases, err := pFlags.GetStringSlice("user")
		if err != nil {
			return err
		}
		langExtensions, err := pFlags.GetStringSlice("lang")
		if err != nil {
			return err
		}
		silent, err := pFlags.GetBool("silent")
		if err != nil {
			return err
		}

		query, _ := pFlags.GetString("query")

		retries, err := pFlags.GetInt("retry")
		if err != nil {
			return err
		}

		users := make(utils.AliasMap[types.User])

		err = users.Parse(userAliases)
		if err != nil {
			return err
		}

		extensions := make(utils.AliasMap[types.Language])

		err = extensions.Parse(langExtensions)
		if err != nil {
			return err
		}

		var g types.Stats

		switch types.GitServer(server) {
		case types.Gitlab:
			gitlab.SetRetries(retries)
			g = gitlab.New(
				host,
				token,
				gitlab.WithRateLimit(rateLimit),
				gitlab.WithUsers(users.ToSlice(models.NewUser)...),
				gitlab.WithLanguages(extensions.ToSlice(models.NewLanguage)...),
				gitlab.WithQuery(query),
				gitlab.WithContext(cmd.Context()),
				gitlab.WithProgress(!silent),
			)
		}

		stats := models.NewStats(g)

		if err := g.Err(); err != nil {
			return err
		}

		format, err := pFlags.GetString("format")
		if err != nil {
			return err
		}

		switch types.Format(format) {
		case types.Json:
			fmt.Println(json.Marshal(&stats))
		case types.Txt:
			fmt.Println(stats.String())
		default:
			return fmt.Errorf("unknown format: %s", format)
		}

		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		pFlags := cmd.PersistentFlags()
		server, err := pFlags.GetString("server")
		if err != nil {
			return err
		}

		switch types.GitServer(server) {
		case types.Gitlab, types.GitHub:
			if err := bindToEnv(cmd, server, "token", "host"); err != nil {
				return err
			}
			if v, err := pFlags.GetString("host"); err != nil {
				return err
			} else {
				u, err := url.Parse(v)
				if err != nil {
					return err
				}

				if u.Scheme == "" {
					u.Scheme = "https"
				}

				err = pFlags.Set("host", u.String())
				if err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid Git server %q", server)
		}

		verbosity, err := pFlags.GetInt("verbosity")
		if err != nil {
			return err
		}

		logrus.SetLevel(logrus.Level(verbosity))

		return nil
	},
}

func init() {
	pFlags := cmd.PersistentFlags()

	pFlags.StringSliceP("user", "u", []string{}, "User aliases in form email:alias")
	pFlags.StringSliceP("lang", "l", []string{}, "Language file extensions in form lang:extension")
	pFlags.StringP("server", "s", "gitlab", "Git server type")
	pFlags.StringP("token", "t", "", "Git server authentication token")
	pFlags.StringP("host", "H", "", "Git server host")
	pFlags.StringP("format", "f", "txt", "Output format")
	pFlags.StringP("query", "q", "", "Projects query")
	pFlags.IntP("retry", "r", 5, "Git server call retries")
	pFlags.IntP("rate", "R", 50, "Git server rate limit")
	pFlags.IntP("verbosity", "v", int(logrus.GetLevel()), "Verbosity level")
	pFlags.BoolP("silent", "S", false, "Don't output progress")
	pFlags.StringP("exclude", "E", "", "Regex for excluding projects")
}
