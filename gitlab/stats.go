package gitlab

import (
	"context"
	"errors"
	"fmt"
	"github.com/gaarutyunov/gitstat/models"
	"github.com/gaarutyunov/gitstat/types"
	"github.com/gaarutyunov/gitstat/utils"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/time/rate"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

type (
	Stats struct {
		ctx         context.Context
		baseURL     *url.URL
		token       string
		query       string
		client      *gitlab.Client
		so          sync.Once
		langs       []types.Language
		langByExt   map[string]types.Language
		userAliases map[string][]string
		userByAlias map[string]types.User
		counter     map[types.User]types.PerLanguageCounter
		se          sync.Once
		err         error
		sem         chan struct{}
		retries     int
		rl          *rate.Limiter
		progress    func(n int64) *progressbar.ProgressBar
		exclude     *regexp.Regexp
	}

	MapLanguageCounter map[types.Language]*atomic.Int64
)

var defaultUser = models.NewUser("other", nil)
var mx sync.Mutex

func makeMapLanguageCounter(keys []types.Language) types.PerLanguageCounter {
	m := make(MapLanguageCounter, len(keys))

	for _, key := range keys {
		m[key] = &atomic.Int64{}
	}

	return m
}

func (m MapLanguageCounter) PerLanguage() (res map[types.Language]int) {
	res = make(map[types.Language]int, len(m))

	for language, counter := range m {
		res[language] = int(counter.Load())
	}

	return
}

func (m MapLanguageCounter) Total() (total int) {
	for _, counter := range m {
		total += int(counter.Load())
	}

	return
}

func (s *Stats) Err() error {
	return s.err
}

type Option func(*Stats)

func New(baseURL, token string, opts ...Option) *Stats {
	g := &Stats{
		ctx:         context.Background(),
		baseURL:     utils.Must(url.Parse(baseURL)),
		token:       token,
		userAliases: make(map[string][]string),
		userByAlias: make(map[string]types.User),
		counter:     make(map[types.User]types.PerLanguageCounter),
		langByExt:   make(map[string]types.Language),
		rl:          rate.NewLimiter(50, 1),
		progress: func(n int64) *progressbar.ProgressBar {
			return progressbar.Default(n)
		},
	}

	for _, opt := range opts {
		opt(g)
	}

	g.client = utils.Must(gitlab.NewClient(
		token,
		gitlab.WithBaseURL(baseURL),
		gitlab.WithCustomLimiter(g.rl),
	))

	return g
}

func (s *Stats) count() {
	projectCh := make(chan *gitlab.Project)
	var totalRepos, processedRepos atomic.Int64

	var incCh chan struct{}
	var decCh chan struct{}
	doneCh := make(chan struct{})

	go func() {
		for repo := range projectCh {
			if s.exclude != nil && s.exclude.MatchString(repo.PathWithNamespace) {
				logrus.Debugf("repository %s doesn't match pattern %s, skipping", repo.PathWithNamespace, s.exclude)
				continue
			}

			totalRepos.Add(1)

			go func() {
				select {
				case <-s.ctx.Done():
					return
				default:
				}

				err := s.processRepo(repo)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						s.se.Do(func() {
							s.err = err
						})
						doneCh <- struct{}{}
						close(doneCh)
						return
					}
					logrus.Error(err)
					s.decProcessed(&totalRepos, decCh)
					return
				}

				s.incProcessed(&processedRepos, incCh)

				if processedRepos.CompareAndSwap(totalRepos.Load(), 0) {
					doneCh <- struct{}{}
					close(doneCh)
				}
			}()
		}
	}()

	if err := s.getUsers(); err != nil {
		s.se.Do(func() {
			s.err = err
		})
		return
	}

	if err := s.getRepos(projectCh); err != nil {
		s.se.Do(func() {
			s.err = err
		})
		close(projectCh)
		return
	} else {
		close(projectCh)
	}

	if s.progress != nil {
		incCh = make(chan struct{})
		decCh = make(chan struct{})

		defer func() {
			close(incCh)
			close(decCh)
		}()

		bar := s.progress(totalRepos.Load())

		go func() {
			for range incCh {
				select {
				case <-s.ctx.Done():
					return
				default:
				}
				bar.Add(1)
			}
		}()

		go func() {
			for range decCh {
				select {
				case <-s.ctx.Done():
					return
				default:
				}
				bar.ChangeMax64(totalRepos.Load())
			}
		}()
	}

	<-doneCh
}

func (s *Stats) incProcessed(counter *atomic.Int64, ch chan<- struct{}) {
	counter.Add(1)
	if ch != nil {
		ch <- struct{}{}
	}
}

func (s *Stats) decProcessed(counter *atomic.Int64, ch chan<- struct{}) {
	counter.Add(-1)
	if ch != nil {
		ch <- struct{}{}
	}
}

func (s *Stats) getRepos(projectCh chan<- *gitlab.Project) error {
	opts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Search: &s.query,
	}

	for {
		pp, res, err := s.client.Projects.ListProjects(opts, gitlab.WithContext(s.ctx))
		if err != nil {
			return errors.Join(errors.New("error listing projects"), err)
		}

		for _, project := range pp {
			projectCh <- project
		}

		if res.CurrentPage == res.TotalPages {
			break
		} else {
			opts.Page = res.NextPage
		}
	}

	return nil
}

func (s *Stats) getUsers() error {
	opts := &gitlab.ListUsersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		users, res, err := s.client.Users.ListUsers(opts, gitlab.WithContext(s.ctx))
		if err != nil {
			return err
		}

		for _, u := range users {
			user := NewUser(u, s.userAliases[u.Email])
			s.counter[user] = makeMapLanguageCounter(s.langs)

			for _, alias := range user.GetAliases() {
				s.userByAlias[alias] = user
			}

			s.userByAlias[user.GetEmail()] = user
		}

		if res.CurrentPage == res.TotalPages {
			break
		} else {
			opts.Page = res.NextPage
		}
	}

	s.counter[defaultUser] = makeMapLanguageCounter(s.langs)

	return nil
}

func (s *Stats) processRepo(repo *gitlab.Project) error {
	opts := &gitlab.ListTreeOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Recursive: gitlab.Ptr(true),
	}

	var totalFiles, processedFiles atomic.Int64
	doneCh := make(chan struct{})

	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		tree, res, err := s.client.Repositories.ListTree(repo.ID, opts, gitlab.WithContext(s.ctx))
		if err != nil {
			if errors.Is(err, gitlab.ErrNotFound) {
				logrus.Debugf("empty tree for repo %s", repo.PathWithNamespace)
				return nil
			}
			return errors.Join(fmt.Errorf("error listing repo tree for project %s", repo.PathWithNamespace), err)
		}

		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		for _, node := range tree {
			if node.Type != "blob" {
				continue
			}

			ext := filepath.Ext(node.Path)

			lang, ok := s.langByExt[ext]
			if !ok {
				logrus.Debugf("skipping file with extension %s", ext)
				continue
			}

			totalFiles.Add(1)

			select {
			case <-s.ctx.Done():
				return s.ctx.Err()
			default:
			}

			go func(path string, lang types.Language) {
				select {
				case <-s.ctx.Done():
					return
				default:
				}

				blame, _, err := s.client.RepositoryFiles.GetFileBlame(
					repo.ID,
					path,
					&gitlab.GetFileBlameOptions{
						Ref: gitlab.Ptr(repo.DefaultBranch),
					},
					gitlab.WithContext(s.ctx),
				)
				if err != nil && !errors.Is(err, context.Canceled) {
					logrus.Debugf("error gettings blame for file %s in repository %s: %v", path, repo.PathWithNamespace, err)
					return
				}

				select {
				case <-s.ctx.Done():
					return
				default:
				}

				for _, blameRange := range blame {
					user, ok := s.userByAlias[blameRange.Commit.CommitterEmail]
					if !ok {
						logrus.Debugf("unknown user %s, using default", blameRange.Commit.CommitterEmail)

						user = defaultUser
					}

					var linesCount int64
					for _, line := range blameRange.Lines {
						if strings.TrimSpace(line) != "" {
							linesCount++
						}
					}

					s.counter[user].(MapLanguageCounter)[lang].Add(linesCount)
				}

				processedFiles.Add(1)

				if processedFiles.CompareAndSwap(totalFiles.Load(), 0) {
					doneCh <- struct{}{}
					close(doneCh)
				}
			}(node.Path, lang)
		}

		if res.CurrentPage == res.TotalPages {
			break
		} else {
			opts.Page = res.NextPage
		}
	}

	if processedFiles.CompareAndSwap(totalFiles.Load(), 0) {
		close(doneCh)
		return nil
	}

	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-doneCh:
			return nil
		}
	}
}

func (s *Stats) PerUser() (res map[types.User]types.PerLanguageCounter) {
	s.so.Do(s.count)

	if s.err != nil {
		return
	}

	res = make(map[types.User]types.PerLanguageCounter)

	for user, counter := range s.counter {
		if counter.Total() == 0 {
			continue
		}

		res[user] = counter
	}

	return
}

func (s *Stats) PerLanguage() (res map[types.Language]int) {
	s.so.Do(s.count)

	if s.err != nil {
		return
	}

	res = make(map[types.Language]int, len(s.langs))

	for _, langs := range s.counter {
		for lang, counter := range langs.(MapLanguageCounter) {
			res[lang] += int(counter.Load())
		}
	}

	return
}

func (s *Stats) Total() (total int) {
	s.so.Do(s.count)

	if s.err != nil {
		return
	}

	for _, langs := range s.counter {
		total += langs.Total()
	}

	return total
}
