package git

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	gitp "github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kyma-incubator/reconciler/pkg/reconciler"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Cloner struct {
	repo         *reconciler.Repository
	autoCheckout bool
	auth         *BasicAuth
	repoClient   RepoClient
	logger       *zap.SugaredLogger
}

type BasicAuth struct {
	Username string
	Password string
}

//go:generate mockery --name RepoClient --case=underscore
type RepoClient interface {
	Clone(ctx context.Context, path string, isBare bool, o *git.CloneOptions) (*git.Repository, error)
	Worktree() (*git.Worktree, error)
	ResolveRevisionOrBranchHead(rev gitp.Revision) (*gitp.Hash, error)
	Fetch(o *git.FetchOptions) error
	PlainCheckout(o *git.CheckoutOptions) error
	DefaultBranch() (*gitp.Reference, error)
}

func NewCloner(repoClient RepoClient, repo *reconciler.Repository, autoCheckout bool, auth *BasicAuth, logger *zap.SugaredLogger) (*Cloner, error) {
	return &Cloner{
		repo:         repo,
		autoCheckout: autoCheckout,
		repoClient:   repoClient,
		auth:         auth,
		logger:       logger,
	}, nil
}

// Clone clones the repository from the given remote URL to the given `path` in the local filesystem.
func (r *Cloner) Clone(path string) (*git.Repository, error) {

	var basicAuth *http.BasicAuth
	if r.auth != nil {
		basicAuth = &http.BasicAuth{
			Username: r.auth.Username,
			Password: r.auth.Password,
		}
	}

	return r.repoClient.Clone(context.Background(), path, false, &git.CloneOptions{
		Depth:             0,
		URL:               r.repo.URL,
		NoCheckout:        !r.autoCheckout,
		Auth:              basicAuth,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
}

// Checkout checks out the given revision.
// revision can be 'main', a release version (e.g. 1.4.1), a commit hash (e.g. 34edf09a).
func (r *Cloner) Checkout(rev string, repo *git.Repository) error {
	w, err := r.repoClient.Worktree()
	if err != nil {
		return errors.Wrap(err, "error getting the GIT worktree")
	}

	// hash, err := r.repoClient.ResolveRevision(gitp.Revision(rev))
	var defaultLister refLister = remoteRefLister{}
	var resolver = revisionResolver{url: r.repo.URL, repository: repo, refLister: defaultLister}

	hash, err := resolver.resolveRevision(rev)
	if err != nil {
		msg := fmt.Sprintf("failed to resolve GIT revision '%s'", rev)
		if r.repo.URL != "" {
			msg += fmt.Sprintf(" using repository '%s' ",
				r.repo.URL)
		}
		return errors.Wrap(err, msg)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	})

	if err != nil {
		return errors.Wrap(err, "Error checking out GIT revision")
	}
	return nil
}

func (r *Cloner) CloneAndCheckout(dstPath, rev string) error {
	repo, err := r.Clone(dstPath)
	if err != nil {
		return errors.Wrapf(err, "Error downloading Git repository (%s)", r.repo)
	}
	if rev == "" {
		head, err := repo.Head()
		if err != nil {
			return err
		}
		rev = head.Hash().String()
	}
	return r.Checkout(rev, repo)
}

func (r *Cloner) FetchAndCheckout(path, version string) error {
	gitClient, err := NewClientWithPath(path)
	if err != nil {
		return err
	}
	var basicAuth *http.BasicAuth
	if r.auth != nil {
		basicAuth = &http.BasicAuth{
			Username: r.auth.Username,
			Password: r.auth.Password,
		}
	}
	err = gitClient.Fetch(&git.FetchOptions{
		Auth:       basicAuth,
		RemoteName: "origin",
	})
	if err != nil {
		return err
	}
	if version != "" {
		defaultBranch, err := gitClient.DefaultBranch()
		if err != nil {
			return err
		}
		return gitClient.PlainCheckout(&git.CheckoutOptions{
			Hash: defaultBranch.Hash(),
		})

	}
	return nil
}

func (r *Cloner) ResolveRevisionOrBranchHead(rev string) (*gitp.Hash, error) {
	return r.repoClient.ResolveRevisionOrBranchHead(gitp.Revision(rev))
}
