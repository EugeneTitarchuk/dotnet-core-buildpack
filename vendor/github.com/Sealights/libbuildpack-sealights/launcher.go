package sealights

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

const AgentName = "SL.DotNet.dll"
const DefaultAgentMode = "testListener"

type Launcher struct {
	Log       *libbuildpack.Logger
	Options   *SealightsOptions
	AgentDir  string
	DotNetDir string
}

func NewLauncher(log *libbuildpack.Logger, options *SealightsOptions, agentInstallationDir string, dotnetInstallationDir string) *Launcher {
	return &Launcher{Log: log, Options: options, AgentDir: agentInstallationDir, DotNetDir: dotnetInstallationDir}
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

//dotnet SL.DotNet.dll testListener --logAppendFile true --logFilename /tmp/collector.log --tokenFile /tmp/sltoken.txt --buildSessionIdFile /tmp/buildsessionid.txt --target dotnet --workingDir /tmp/app --profilerLogDir /tmp/ --profilerLogLevel 7 --targetArgs \"test app.dll\"
func (la *Launcher) buildCommandLine(targetProgram string, targetArgs string) string {

	var sb strings.Builder
	options := la.Options

	agent := filepath.Join(la.AgentDir, AgentName)
	dotnetCli := filepath.Join(la.DotNetDir, "dotnet")

	agentMode := DefaultAgentMode
	if options.Mode != "" {
		agentMode = options.Mode
	}

	sb.WriteString(fmt.Sprintf("%s %s %s", dotnetCli, agent, agentMode))

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

	if options.ProfilerLogDir != "" {
		sb.WriteString(fmt.Sprintf(" --profilerLogDir %s", options.ProfilerLogDir))
	}

	if options.ProfilerLogLevel != "" {
		sb.WriteString(fmt.Sprintf(" --profilerLogLevel %s", options.ProfilerLogLevel))
	}

	if options.Tags != "" {
		sb.WriteString(fmt.Sprintf(" --tags %s", options.Tags))
	}

	if options.Tools != "" {
		sb.WriteString(fmt.Sprintf(" --tools %s", options.Tools))
	}

	if options.IgnoreCertificateErrors == "true" {
		sb.WriteString(" --ignoreCertificateErrors true")
	}

	if options.NotCli == "true" {
		sb.WriteString(" --notCli true")
	}

	if options.Proxy != "" {
		sb.WriteString(fmt.Sprintf(" --proxy %s", options.Proxy))
		sb.WriteString(fmt.Sprintf(" --proxyUsername %s", options.ProxyUsername))
		sb.WriteString(fmt.Sprintf(" --proxyPassword %s", options.ProxyPassword))
	}

	sb.WriteString(" --workingDir ${PWD}")

	if agentMode == DefaultAgentMode {
		targetProgram = dotnetCli
		targetArgs = fmt.Sprintf("test %s %s", targetProgram, targetArgs)
	}

	if (strings.HasPrefix(targetArgs, "--")){
		targetArgs = fmt.Sprintf(" %s", targetArgs)
	}

	sb.WriteString(fmt.Sprintf(" --target %s --targetArgs \"%s\"", targetProgram, targetArgs))

	return sb.String()
}
