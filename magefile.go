//go:build mage
// +build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mt-sre/go-ci/command"
)

var Aliases = map[string]interface{}{
	"lint":     All.Lint,
	"generate": All.Generate,
	"test": All.Test,
}

type All mg.Namespace

func (All) Lint(ctx context.Context) {
	mg.SerialCtxDeps(
		ctx,
		All.Generate,
		Check.Tidy,
		Check.Verify,
		Check.Lint,
		Check.Dirty,
	)
}

func (All) Test(ctx context.Context) {
	mg.CtxDeps(
		ctx,
		All.Generate,
		Test.Unit,
		Test.Integration,
	)
}

func (All) Generate(ctx context.Context) {
	mg.CtxDeps(
		ctx,
		Generate.Manifests,
		Generate.Boilerplate,
	)
}

var _depBin = filepath.Join(_dependencyDir, "bin")

var _dependencyDir = func() string {
	if dir, ok := os.LookupEnv("DEPENDENCY_DIR"); ok {
		return dir
	}

	return filepath.Join(_projectRoot, ".cache", "dependencies")
}()

var _projectRoot = func() string {
	if root, ok := os.LookupEnv("PROJECT_ROOT"); ok {
		return root
	}

	topLevel := git(command.WithArgs{"rev-parse", "--show-toplevel"})

	if err := topLevel.Run(); err != nil || !topLevel.Success() {
		panic("failed to get working directory")
	}

	return strings.TrimSpace(topLevel.Stdout())
}()

var git = command.NewCommandAlias("git")

type Deps mg.Namespace

func (Deps) UpdateControllerGen(ctx context.Context) error {
	return updateGODependency(ctx, "sigs.k8s.io/controller-tools/cmd/controller-gen")
}

func (Deps) UpdateGinkgo(ctx context.Context) error {
	return updateGODependency(ctx, "github.com/onsi/ginkgo/v2/ginkgo")
}

func (Deps) UpdateGolangCILint(ctx context.Context) error {
	return updateGODependency(ctx, "github.com/golangci/golangci-lint/cmd/golangci-lint")
}

func (Deps) UpdateSetupEnvtest(ctx context.Context) error {
	return updateGODependency(ctx, "sigs.k8s.io/controller-runtime/tools/setup-envtest")
}

func updateGODependency(ctx context.Context, src string) error {
	if err := setupDepsBin(); err != nil {
		return fmt.Errorf("creating dependencies bin directory: %w", err)
	}

	toolsDir := filepath.Join(_projectRoot, "tools")

	tidy := gocmd(
		command.WithArgs{"mod", "tidy", "-compat=1.17"},
		command.WithWorkingDirectory(toolsDir),
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := tidy.Run(); err != nil {
		return fmt.Errorf("starting to tidy tools dir: %w", err)
	}

	if !tidy.Success() {
		return fmt.Errorf("tidying tools dir: %w", tidy.Error())
	}

	install := gocmd(
		command.WithArgs{"install", src},
		command.WithWorkingDirectory(toolsDir),
		command.WithCurrentEnv(true),
		command.WithEnv{"GOBIN": _depBin},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := install.Run(); err != nil {
		return fmt.Errorf("starting to install command from source %q: %w", src, err)
	}

	if !install.Success() {
		return fmt.Errorf("installing command from source %q: %w", src, install.Error())
	}

	return nil
}

var gocmd = command.NewCommandAlias(mg.GoCmd())

func setupDepsBin() error {
	return os.MkdirAll(_depBin, 0o774)
}

// Removes any existing dependency binaries
func (Deps) Clean() error {
	return sh.Rm(_depBin)
}

type Check mg.Namespace

// Runs linter against source code.
func (Check) Lint(ctx context.Context) error {
	mg.CtxDeps(
		ctx,
		Deps.UpdateGolangCILint,
	)

	run := golangci(
		command.WithArgs{"run", "-v", "--fix"},
		command.WithContext{Context: ctx},
	)

	if err := run.Run(); err != nil {
		return fmt.Errorf("starting linter: %w", err)
	}

	if run.Success() {
		return nil
	}

	fmt.Fprint(os.Stdout, run.CombinedOutput())

	return fmt.Errorf("running linter: %w", run.Error())
}

var golangci = command.NewCommandAlias(filepath.Join(_depBin, "golangci-lint"))

// Ensures dependencies are correctly updated in the 'go.mod'
// and 'go.sum' files.
func (Check) Tidy(ctx context.Context) error {
	tidy := gocmd(
		command.WithArgs{"mod", "tidy", "-compat=1.17"},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := tidy.Run(); err != nil {
		return fmt.Errorf("starting to tidy go dependencies: %w", err)
	}

	if tidy.Success() {
		return nil
	}

	return fmt.Errorf("tidying go dependencies: %w", tidy.Error())
}

// Ensures package dependencies have not been tampered with since download.
func (Check) Verify(ctx context.Context) error {
	verify := gocmd(
		command.WithArgs{"mod", "verify"},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := verify.Run(); err != nil {
		return fmt.Errorf("starting to verify go dependencies: %w", err)
	}

	if verify.Success() {
		return nil
	}

	return fmt.Errorf("verifying go dependencies: %w", verify.Error())
}

func (Check) Dirty(ctx context.Context) error {
	status := git(
		command.WithArgs{"status", "--porcelain"},
	)

	if err := status.Run(); err != nil {
		return fmt.Errorf("starting to check git status: %w", err)
	}

	if !status.Success() {
		return fmt.Errorf("checking git status: %w", status.Error())
	}

	if out := status.Stdout(); out != "" {
		return errors.New("repo is dirty")
	}

	return nil
}

type Test mg.Namespace

// Runs unit tests.
func (Test) Unit(ctx context.Context) error {
	mg.CtxDeps(
		ctx,
		All.Generate,
	)

	test := gocmd(
		command.WithArgs{"test", "-race", "-v", "./cmd/...", "./internal/..."},
		command.WithCurrentEnv(true),
		command.WithEnv{
			"CGO_ENABLED": "1",
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := test.Run(); err != nil {
		return fmt.Errorf("starting unit tests: %w", err)
	}

	if test.Success() {
		return nil
	}

	return fmt.Errorf("running unit tests: %w", test.Error())
}

// Runs integration tests.
func (Test) Integration(ctx context.Context) error {
	mg.CtxDeps(
		ctx,
		Deps.UpdateGinkgo,
		Deps.UpdateSetupEnvtest,
	)

	setup := setupEnvtest(
		command.WithArgs{
			"use", "-p", "path", "--bin-dir=" + _depBin, "1.20.x!",
		},
	)

	if err := setup.Run(); err != nil {
		return fmt.Errorf("starting to setup envtest: %w", err)
	}

	if !setup.Success() {
		return fmt.Errorf("setting up envtest: %w", setup.Error())
	}

	fmt.Println(setup.Stdout())

	test := ginkgo(
		command.WithArgs{
			"-r",
			"--randomize-all",
			"--randomize-suites",
			"--fail-on-pending",
			"--keep-going",
			"--trace",
			"--no-color",
			"-v",
			"integration",
		},
		command.WithCurrentEnv(true),
		command.WithEnv{
			"KUBEBUILDER_ASSETS": setup.Stdout(),
			"XDG_CACHE_HOME": filepath.Join(_projectRoot, ".cache"),
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := test.Run(); err != nil {
		return fmt.Errorf("starting integration tests: %w", err)
	}

	if test.Success() {
		return nil
	}

	return fmt.Errorf("running integration tests: %w", test.Error())
}

var (
	ginkgo       = command.NewCommandAlias(filepath.Join(_depBin, "ginkgo"))
	setupEnvtest = command.NewCommandAlias(filepath.Join(_depBin, "setup-envtest"))
)

type Generate mg.Namespace

// Generates manifests.
func (Generate) Manifests(ctx context.Context) error {
	mg.CtxDeps(
		ctx,
		Deps.UpdateControllerGen,
	)

	gen := controllerGen(
		command.WithArgs{
			"crd:crdVersions=v1",
			"rbac:roleName=reference-addon",
			"output:crd:artifacts:config=config/deploy",
			`paths="./apis/..."`,
			`paths="./cmd/..."`,
			`paths="./internal/..."`,
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := gen.Run(); err != nil {
		return fmt.Errorf("starting to generate manifests: %w", err)
	}

	if gen.Success() {
		return nil
	}

	return fmt.Errorf("generating manifests: %w", gen.Error())
}

// Generates objects.
func (Generate) Boilerplate(ctx context.Context) error {
	mg.CtxDeps(
		ctx,
		Deps.UpdateControllerGen,
	)

	gen := controllerGen(
		command.WithArgs{
			"object", `paths="./apis/..."`,
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := gen.Run(); err != nil {
		return fmt.Errorf("starting to generate objects: %w", err)
	}

	if gen.Success() {
		return nil
	}

	return fmt.Errorf("generating objects: %w", gen.Error())
}

var controllerGen = command.NewCommandAlias(filepath.Join(_depBin, "controller-gen"))
