//go:build mage
// +build mage

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"

	"github.com/magefile/mage/mg" // mg contains helpful utility functions, like Deps
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

// A build step that requires additional params, or platform specific steps for example
func Build() error {
	mg.Deps(InstallDeps)
	fmt.Println("Building...")
	cmd := exec.Command("go", "build", "-o", "zengo", ".")
	return cmd.Run()
}

func asyncCommand(ctx context.Context, stepName string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stderr, "%s: %s\n", stepName, line)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stdout, "%s: %s\n", stepName, line)
		}
	}()

	return cmd.Wait()
}

func LiveServer(ctx context.Context) error {
	fmt.Println("Starting air")
	return asyncCommand(ctx, "server", "go", "run", "github.com/cosmtrek/air@v1.51.0",
		"--build.exclude_dir", "node_modules",
		"--build.include_ext", "go",
		"--build.stop_on_error", "false",
		"--misc.clean_on_exit", "true")
}

func LiveTempl(ctx context.Context) error {
	fmt.Println("Starting templ")
	return asyncCommand(
		ctx,
		"templ",
		"templ",
		"generate",
		"--watch",
		"--proxy=http://localhost:3000",
		"--open-browser=false",
		"-v",
	)
}

func LiveTailwind(ctx context.Context) error {
	fmt.Println("Starting Tailwind")
	return asyncCommand(ctx, "tailwindcss", "npx", "tailwindcss", "-i",
		"./static/css/input.css",
		"-o",
		"./static/css/style.css",
		"--watch=always",
	)
}

func LiveSyncAssets(ctx context.Context) error {
	return asyncCommand(ctx, "sync-assets", "go", "run", "github.com/cosmtrek/air@v1.51.0",
		"--build.cmd", "templ generate --notify-proxy",
		"--build.bin", "true",
		"--build.delay", "100",
		"--build.exclude_dir", "",
		"--build.include_dir", "static",
		"--build.include_ext", "js,css",
	)
}

func Dev() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	errCh := make(chan error)

	wg := sync.WaitGroup{}
	wg.Add(4)
	go func() {
		defer wg.Done()
		if err := LiveTailwind(ctx); err != nil {
			<-errCh
		}
	}()
	go func() {
		defer wg.Done()
		if err := LiveTempl(ctx); err != nil {
			<-errCh
		}
	}()
	go func() {
		defer wg.Done()
		if err := LiveServer(ctx); err != nil {
			<-errCh
		}
	}()
	go func() {
		defer wg.Done()
		if err := LiveSyncAssets(ctx); err != nil {
			<-errCh
		}
	}()

	select {
	case <-ctx.Done():
		wg.Wait()
	case err := <-errCh:
		fmt.Println(err)
		cancel()
		wg.Wait()
		return err
	}

	return nil
}

// A custom install step if you need your bin someplace other than go/bin
func Install() error {
	mg.Deps(Build)
	fmt.Println("Installing...")
	return os.Rename("./MyApp", "/usr/bin/MyApp")
}

// Manage your deps, or running package managers.
func InstallDeps() error {
	fmt.Println("Installing Deps...")
	cmd := exec.Command("go", "get", "github.com/stretchr/piglatin")
	return cmd.Run()
}

// Clean up after yourself
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("MyApp")
}
