package git

import (
	"context"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/kyma-incubator/reconciler/pkg/logger"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kyma-incubator/reconciler/pkg/reconciler"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/git/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/alcortesm/tgz"
	"github.com/go-git/go-git/v5"
	gitp "github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
)

// TestCloneRepo tests CloneAndCheckout function that is provided with a dummy git repository (no actual cloning is performed)
// The repo has the following commits
// 1. Add README (tagged with 1.0.0)
// 2. Update README (tagged with 2.0.0 - HEAD)
func TestCloneRepo(t *testing.T) {
	localRepoRootPath, err := tgz.Extract("testdata/repo.tgz")
	defer func() {
		require.NoError(t, os.RemoveAll(localRepoRootPath))
	}()
	require.NoError(t, err)
	require.NotEmpty(t, localRepoRootPath)

	repo, err := git.PlainOpen(path.Join(localRepoRootPath, "repo"))
	require.NoError(t, err)

	var refs []*gitp.Reference
	iter, err := repo.References()
	require.NoError(t, err)

	err = iter.ForEach(func(r *gitp.Reference) error {
		refs = append(refs, r)
		return nil
	})
	require.NoError(t, err)

	clonerMock := &mocks.RepoClient{}

	repoURL := "github.com/foo"
	r := reconciler.Repository{URL: repoURL}
	options := git.CloneOptions{
		Depth:             0,
		URL:               r.URL,
		NoCheckout:        false,
		Auth:              nil,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}
	clonerMock.On("Clone",
		context.Background(), "bar/baz", false, &options).
		Return(repo, nil)
	clonerMock.On("Worktree").
		Return(repo.Worktree())
	clonerMock.On("ResolveRevision",
		gitp.Revision("1.0.0")).
		Return(repo.ResolveRevision("1.0.0"))
	cloner, _ := NewCloner(clonerMock, &r, nil, logger.NewLogger(true))

	headRef, err := repo.Head()
	require.NoError(t, err)

	commit, err := repo.CommitObject(headRef.Hash())
	require.NoError(t, err)
	require.Equal(t, "Update README\n", commit.Message)

	err = cloner.CloneAndCheckout("bar/baz", "1.0.0")
	require.NoError(t, err)

	headRef, err = repo.Head()
	require.NoError(t, err)

	commit, err = repo.CommitObject(headRef.Hash())
	require.NoError(t, err)
	require.Equal(t, "Add README\n", commit.Message)

	t.Run("Should add auth data if token namespace set and token exists", func(t *testing.T) {
		expectedOpts := &git.CloneOptions{
			Depth:      0,
			URL:        repoURL,
			NoCheckout: false,
			Auth: &http.BasicAuth{
				Username: "xxx",
				Password: "token",
			},
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		}
		clonerMock.On("Clone",
			context.Background(),
			"/test",
			false,
			mock.MatchedBy(func(opts *git.CloneOptions) bool { return reflect.DeepEqual(*opts, *expectedOpts) }),
		).Return(repo, nil)

		cloner, _ := NewCloner(
			clonerMock,
			&reconciler.Repository{URL: repoURL, TokenNamespace: "default"},
			&BasicAuth{Username: "xxx", Password: "token"},
			logger.NewLogger(true),
		)

		_, err := cloner.Clone("/test")
		assert.NoError(t, err)
	})
}

// func TestTokenRead(t *testing.T) {
// 	t.Parallel()

// 	t.Run("Should read correct token secret", func(t *testing.T) {
// 		repo := reconciler.Repository{
// 			URL:            "https://localhost",
// 			TokenNamespace: "default",
// 		}

// 		cloner := Cloner{
// 			repo:       &repo,
// 			repoClient: nil,
// 			logger:     logger.NewLogger(true),
// 		}

// 		assert.Equal(t, &http.BasicAuth{
// 			Username: "xxx",
// 			Password: "tokenValue",
// 		}, auth)
// 	})

// 	t.Run("Should ignore error when token secret not found", func(t *testing.T) {
// 		repo := reconciler.Repository{
// 			URL:            "https://localhost",
// 			TokenNamespace: "default",
// 		}

// 		cloner := Cloner{
// 			repo:       &repo,
// 			repoClient: nil,
// 			logger:     logger.NewLogger(true),
// 		}

// 		_, err := cloner.buildAuth()

// 		assert.NoError(t, err)
// 	})

// 	t.Run("Should ignore error when clientset not set", func(t *testing.T) {
// 		repo := reconciler.Repository{
// 			URL:            "https://localhost",
// 			TokenNamespace: "default",
// 		}

// 		cloner := Cloner{
// 			repo:       &repo,
// 			repoClient: nil,
// 			logger:     logger.NewLogger(true),
// 		}

// 		_, err := cloner.buildAuth()

// 		assert.NoError(t, err)
// 	})

// 	t.Run("Should ignore error when TokenNamespace not set", func(t *testing.T) {
// 		repo := reconciler.Repository{
// 			URL:            "https://localhost",
// 			TokenNamespace: "",
// 		}

// 		cloner := Cloner{
// 			repo:       &repo,
// 			repoClient: nil,
// 			logger:     logger.NewLogger(true),
// 		}

// 		_, err := cloner.buildAuth()

// 		assert.NoError(t, err)
// 	})

// 	t.Run("Should parse URL", func(t *testing.T) {
// 		assertParsed(t, "localhost", "localhost/path")
// 		assertParsed(t, "localhost", "localhost")
// 		assertParsed(t, "localhost", "localhost:8080")
// 		assertParsed(t, "localhost", "http://localhost:8080")
// 		assertParsed(t, "localhost", "www.localhost:8080")
// 		assertParsed(t, "localhost", "https://www.localhost:8080")
// 		assertParsed(t, "192.168.1.2", "192.168.1.2")
// 		assertParsed(t, "192.168.1.2", "192.168.1.2:8080")
// 	})
// }

// func assertParsed(t *testing.T, expected string, url string) {
// 	key, err := mapSecretKey(url)
// 	assert.NoError(t, err)
// 	assert.Equal(t, expected, key)
// }
