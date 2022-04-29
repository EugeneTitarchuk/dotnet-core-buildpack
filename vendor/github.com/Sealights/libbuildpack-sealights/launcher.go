package sealights

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

const Procfile = "Procfile"
const ManifestFile = "manifest.yml"
const AgentName = "SL.DotNet.dll"
const AgentMode = "testListener"

type Manifest struct {
	Applications []struct {
		Name    string `yaml:"name"`
		Command string `yaml:"command"`
	} `yaml:"applications"`
}

type Launcher struct {
	Log      *libbuildpack.Logger
	Options  *SealightsOptions
	AgentDir string
}

func NewLauncher(log *libbuildpack.Logger, options *SealightsOptions, agentInstallationDir string) *Launcher {
	return &Launcher{Log: log, Options: options, AgentDir: agentInstallationDir}
}

func (la *Launcher) ModifyStartParameters(stager *libbuildpack.Stager) error {
	if _, err := os.Stat(filepath.Join(stager.BuildDir(), Procfile)); err == nil {
		la.Log.Info("Sealights. Modify start command in procfile")
		return la.setApplicationStartInProcfile(stager)
	} else if _, err := os.Stat(filepath.Join(stager.BuildDir(), ManifestFile)); err == nil {
		la.Log.Info("Sealights. Modify start command in manifest.yml")
		return la.setApplicationStartInManifest(stager)
	} else {
		return fmt.Errorf("Failed to detect launch command type")
	}
}

func (la *Launcher) setApplicationStartInProcfile(stager *libbuildpack.Stager) error {
	bytes, err := ioutil.ReadFile(filepath.Join(stager.BuildDir(), Procfile))
	if err != nil {
		la.Log.Error("Sealights. Failed to read file '%s'", Procfile)
		return err
	}

	// we suppose that format is "web: dotnet <application>"
	startCommand := la.updateStartCommand(string(bytes), stager.BuildDir())

	err = ioutil.WriteFile(filepath.Join(stager.BuildDir(), Procfile), []byte(startCommand), 0755)
	if err != nil {
		la.Log.Error("Sealights. Failed to update %s, error: %s", Procfile, err.Error())
		return err
	}

	return nil
}

func (la *Launcher) setApplicationStartInManifest(stager *libbuildpack.Stager) error {
	yaml := &libbuildpack.YAML{}
	manifest, err := la.readManifestFile(stager, yaml)
	if err != nil {
		return err
	}

	originalCommand := manifest.Applications[0].Command

	// we suppose that format is "start: dotnet <application>"
	startCommand := la.updateStartCommand(originalCommand, stager.BuildDir())

	manifest.Applications[0].Command = startCommand
	err = yaml.Write(filepath.Join(stager.BuildDir(), ManifestFile), manifest)
	if err != nil {
		la.Log.Error("Sealights. Failed to update %s, error: %s", ManifestFile, err.Error())
		return err
	}

	return nil
}

func (la *Launcher) readManifestFile(stager *libbuildpack.Stager, yaml *libbuildpack.YAML) (Manifest, error) {
	var manifest Manifest
	err := yaml.Load(filepath.Join(stager.BuildDir(), ManifestFile), &manifest)
	if err != nil {
		la.Log.Error("Sealights. Failed to read %s error: %s", ManifestFile, err.Error())
		return manifest, err

	}
	return manifest, nil
}

func (la *Launcher) updateStartCommand(originalCommand string, workingDir string) string {

	parts := strings.SplitAfter(originalCommand, "dotnet")

	newCmd := parts[0] + la.buildCommandLine(parts[1], workingDir)

	la.Log.Info("Sealights. New start command is: %s", newCmd)

	return newCmd
}

//SL.DotNet.dll testListener --logAppendFile true --logFilename /home/eugene/dev/Sealights/Logs/profilerlogs/newtonsoft_collector.log --tokenFile /home/eugene/dev/Sealights/Environment/sltoken.txt --buildSessionIdFile /home/eugene/dev/Sealights/Environment/buildsessionid.txt --target dotnet --workingDir /home/eugene/dev/Sealights/Samples/Newtonsoft.Json-13.0.1/Src/Newtonsoft.Json.Tests/bin/Debug/net5.0 --profilerLogDir /home/eugene/dev/Sealights/Logs/profilerlogs/ --profilerLogLevel 7 --targetArgs \"test Newtonsoft.Json.Tests.dll --logger:console;verbosity=detailed \"
func (la *Launcher) buildCommandLine(target string, workingDir string) string {

	var sb strings.Builder
	options := la.Options

	agent := filepath.Join(la.AgentDir, AgentName)

	sb.WriteString(fmt.Sprintf("%s %s", agent, AgentMode))

	if options.TokenFile != "" {
		sb.WriteString(fmt.Sprintf(" --tokenfile %s", options.TokenFile))
	} else {
		sb.WriteString(fmt.Sprintf(" --token %s", options.Token))
	}

	if options.BsIdFile != "" {
		sb.WriteString(fmt.Sprintf(" --buildSessionIdFile %s", options.BsIdFile))
	} else {
		sb.WriteString(fmt.Sprintf(" --buildSessionId %s", options.BsId))
	}

	sb.WriteString(fmt.Sprintf(" --workingDir %s", workingDir))
	sb.WriteString(fmt.Sprintf(" --target dotnet --targetArgs \"%s\"", target))

	return sb.String()
}
