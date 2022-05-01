package integration_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func testSealights(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		app      *cutlass.App
		services []string
	)

	it.Before(func() {
		app = cutlass.New(filepath.Join(settings.FixturesPath, "source_apps", "source_6.0"))
		app.SetEnv("BP_DEBUG", "true")
		PushAppAndConfirm(t, app)
	})

	it.After(func() {
		app = DestroyApp(t, app)

		for _, service := range services {
			command := exec.Command("cf", "delete-service", "-f", service)
			_, err := command.Output()
			Expect(err).NotTo(HaveOccurred())
		}
	})

	context("deploying a .NET 6 app with Sealights service", func() {
		it("checks if Sealights injection was successful", func() {
			service := "sealights-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", service, "-p", "'{\"version\":\"3.0.0-beta.2\",\"token\":\"test\",\"bsId\":\"test\"}'")
			output, err := command.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))
			services = append(services, service)

			command = exec.Command("cf", "bind-service", app.Name, service)
			output, err = command.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			command = exec.Command("cf", "restage", app.Name)
			output, err = command.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			Expect(app.ConfirmBuildpack(settings.Buildpack.Version)).To(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Sealights. Service enabled"))
			Expect(app.Stdout.String()).To(ContainSubstring("Sealights. Agent installed"))
			Expect(app.Stdout.String()).To(ContainSubstring("Sealights. Service is set up"))
		})
	})
}
