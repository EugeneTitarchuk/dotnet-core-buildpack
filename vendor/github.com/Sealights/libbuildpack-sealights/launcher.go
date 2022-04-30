package sealights

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

const AgentName = "SL.DotNet.dll"
const AgentMode = "testListener"

type Launcher struct {
	Log      *libbuildpack.Logger
	Options  *SealightsOptions
	AgentDir string
}

func NewLauncher(log *libbuildpack.Logger, options *SealightsOptions, agentInstallationDir string) *Launcher {
	return &Launcher{Log: log, Options: options, AgentDir: agentInstallationDir}
}

func (la *Launcher) ModifyStartParameters(stager *libbuildpack.Stager) error {
	la.updateAgentPath(stager)

	releaseInfo := NewReleaseInfo(stager.BuildDir())

	startCommand := releaseInfo.GetStartCommand()
	newStartCommand := la.updateStartCommand(startCommand)
	err := releaseInfo.SetStartCommand(newStartCommand)
	if err != nil {
		return err
	}

	la.Log.Info(fmt.Sprintf("Sealights: Start command updated. From '%s' to '%s'", startCommand, newStartCommand))

	return nil
}

func (la *Launcher) updateAgentPath(stager *libbuildpack.Stager) {
	if strings.HasPrefix(la.AgentDir, stager.BuildDir()) {
		clearPath := strings.TrimPrefix(la.AgentDir, stager.BuildDir())
		la.AgentDir = filepath.Join(".", clearPath)
	}
}

func (la *Launcher) updateStartCommand(originalCommand string) string {
	// expected command format:
	// cd ${DEPS_DIR}/0/dotnet_publish && exec ./app --server.urls http://0.0.0.0:${PORT}
	// cd ${DEPS_DIR}/0/dotnet_publish && exec dotnet ./app.dll --server.urls http://0.0.0.0:${PORT}

	parts := strings.SplitAfterN(originalCommand, "exec ", 2)
	command := parts[1]
	target := "exec"
	if strings.HasPrefix(command, "dotnet") {
		target = "dotnet"
		command = strings.TrimPrefix(command, "dotnet")
	}

	newCmd := parts[0] + la.buildCommandLine(target, command)

	return newCmd
}

//SL.DotNet.dll testListener --logAppendFile true --logFilename /home/eugene/dev/Sealights/Logs/profilerlogs/newtonsoft_collector.log --tokenFile /home/eugene/dev/Sealights/Environment/sltoken.txt --buildSessionIdFile /home/eugene/dev/Sealights/Environment/buildsessionid.txt --target dotnet --workingDir /home/eugene/dev/Sealights/Samples/Newtonsoft.Json-13.0.1/Src/Newtonsoft.Json.Tests/bin/Debug/net5.0 --profilerLogDir /home/eugene/dev/Sealights/Logs/profilerlogs/ --profilerLogLevel 7 --targetArgs \"test Newtonsoft.Json.Tests.dll --logger:console;verbosity=detailed \"
func (la *Launcher) buildCommandLine(targetProgram string, targetArgs string) string {

	var sb strings.Builder
	options := la.Options

	agent := filepath.Join(".", "sealights", AgentName)

	sb.WriteString(fmt.Sprintf("dotnet %s %s", agent, AgentMode))

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

	sb.WriteString(" --workingDir $PWD")
	sb.WriteString(fmt.Sprintf(" --target %s --targetArgs \"%s\"", targetProgram, targetArgs))

	return sb.String()
}
