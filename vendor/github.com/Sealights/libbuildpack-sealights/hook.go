package sealights

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

// Command is an interface around libbuildpack.Command. Represents an executor for external command calls. We have it
// as an interface so that we can mock it and use in the unit tests.
type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
}

// SealightsHook implements libbuildpack.Hook. It downloads and install the Dynatrace OneAgent.
type SealightsHook struct {
	libbuildpack.DefaultHook
	Log     *libbuildpack.Logger
	Command Command
}

// NewHook returns a libbuildpack.Hook instance for integrating with Sealights
func NewHook() libbuildpack.Hook {
	return &SealightsHook{
		Log:     libbuildpack.NewLogger(os.Stdout),
		Command: &libbuildpack.Command{},
	}
}

func (h *SealightsHook) BeforeCompile(stager *libbuildpack.Stager) error {
	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		h.Log.Error("Unable to determine buildpack directory: %s", err.Error())
		os.Exit(9)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, h.Log, time.Now())
	if err != nil {
		h.Log.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(10)
	}

	h.Log.Info("%v", manifest.ManifestEntries)

	return nil
}

// AfterCompile downloads and installs the Sealighs agent.
func (h *SealightsHook) AfterCompile(stager *libbuildpack.Stager) error {

	h.Log.Debug("Sealights. Check servicec status...")

	conf := NewConfiguration(h.Log)
	if !conf.UseSealights() {
		h.Log.Debug("Sealights service isn't configured")
		return nil
	}

	h.Log.Info("Sealights. Service enabled")

	installationPath := filepath.Join(stager.BuildDir(), "sealights")
	installer := NewInstaller(h.Log, conf.Value)
	err := installer.InstallAgent(installationPath)
	if err != nil {
		return err
	}

	launcher := NewLauncher(h.Log, conf.Value, installationPath)
	launcher.ModifyStartParameters(stager)

	// Get buildpack version and language

	lang := stager.BuildpackLanguage()
	ver, err := stager.BuildpackVersion()
	if err != nil {
		h.Log.Warning("Failed to get buildpack version: %v", err)
		ver = "unknown"
	}

	h.Log.Info("Sealights. Language: %s. Buildpack version: %s.", lang, ver)

	return nil
}
