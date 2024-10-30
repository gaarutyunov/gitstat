package gitlab

import (
	"context"
	"github.com/gaarutyunov/gitstat/types"
	"github.com/gaarutyunov/gitstat/utils"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/schollz/progressbar/v3"
	"github.com/ybbus/httpretry"
	"golang.org/x/time/rate"
	"regexp"
	"time"
)

func WithLanguages(langs ...types.Language) Option {
	return func(g *Stats) {
		g.langs = langs
		for _, lang := range g.langs {
			for _, ext := range lang.Ext() {
				g.langByExt[ext] = lang
			}
		}
	}
}

func WithUsers(users ...types.User) Option {
	return func(g *Stats) {
		for _, user := range users {
			g.userAliases[user.GetEmail()] = user.GetAliases()
			for _, alias := range user.GetAliases() {
				g.userByAlias[alias] = user
			}
		}
	}
}

func WithRateLimit(n int) Option {
	return func(g *Stats) {
		g.rl = rate.NewLimiter(rate.Limit(n), 1)
	}
}

func WithContext(ctx context.Context) Option {
	return func(g *Stats) {
		g.ctx = ctx
	}
}

func WithProgress(progress bool, option ...progressbar.Option) Option {
	return func(g *Stats) {
		if !progress {
			g.progress = nil
		} else {
			g.progress = func(n int64) *progressbar.ProgressBar {
				if len(option) == 0 {
					return progressbar.Default(n)
				}
				return progressbar.NewOptions64(n, option...)
			}
		}
	}
}

func SetRetries(n int) {
	mx.Lock()
	c := http.NewClient(httpretry.NewDefaultClient(
		httpretry.WithMaxRetryCount(n),
		httpretry.WithBackoffPolicy(httpretry.ExponentialBackoff(1*time.Second, 30*time.Second, 5*time.Second)),
	))
	client.InstallProtocol("https", c)
	client.InstallProtocol("http", c)
	mx.Unlock()
}

func WithQuery(s string) Option {
	return func(stats *Stats) {
		stats.query = s
	}
}

func WithExclude(pattern string) Option {
	return func(g *Stats) {
		g.exclude = utils.Must(regexp.Compile(pattern))
	}
}
