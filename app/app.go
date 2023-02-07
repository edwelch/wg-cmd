package app

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"github.com/andrianbdn/wg-cmd/backend"
	"github.com/andrianbdn/wg-cmd/sysinfo"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

type App struct {
	Settings *Settings
	State    *backend.State
}

func NewApp() *App {
	configureLogger()

	settings, err := readSettings()
	if err != nil {
		fmt.Println("Fatal error when reading settings", err)
		os.Exit(1)
	}

	a := App{
		Settings: settings,
	}

	err = a.LoadInterface(a.Settings.DefaultInterface)
	if err != nil {
		panic(err)
	}

	return &a
}

func (a *App) LoadInterface(ifName string) error {
	a.State = nil

	if ifName == "" {
		return nil
	}
	p := a.interfaceDir(ifName)
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	err := os.Chdir(p)
	if err != nil {
		return fmt.Errorf("can't chdir %s:%w", p, err)
	}

	state, err := backend.ReadState(p, log.New(io.Discard, "", 0))
	if err == nil {
		a.State = state
	}
	return err
}

func (a *App) GenerateWireguardConfig() error {
	configPath := filepath.Join(a.Settings.WireguardDir, a.State.Server.Interface) + ".conf"
	return a.State.GenerateWireguardFile(configPath, false)
}

func (a *App) ValidateIfaceArg(ifName string) string {
	if !regexp.MustCompile(`^wg\d{1,4}$`).MatchString(ifName) {
		return "Interface name should be in form wg<number>"
	}

	p := filepath.Join(a.Settings.WireguardDir, ifName+".conf")
	if _, err := os.Stat(p); err == nil {
		return fmt.Sprintf("Found config for %s at %s. Try a different name.", ifName, a.Settings.WireguardDir)
	}

	p = a.interfaceDir(ifName)
	if _, err := os.Stat(p); err == nil {
		return fmt.Sprintf("Found directory %s at %s. Try a different name.",
			filepath.Base(p),
			a.Settings.WireguardDir)
	}

	if sysinfo.NetworkInterfaceExists(ifName) {
		return fmt.Sprintf("Network interface exists in routing tables. Try a different name.")
	}

	return ""
}

func (a *App) TestDirectories() string {
	dbTest := testIfDirWritable(a.Settings.DatabaseDir)

	if a.Settings.DatabaseDir == a.Settings.WireguardDir || dbTest != "" {
		return dbTest
	}

	return testIfDirWritable(a.Settings.WireguardDir)
}

func (a *App) interfaceDir(i string) string {
	d := "wgc-" + i
	return filepath.Join(a.Settings.DatabaseDir, d)
}

func testIfDirWritable(dir string) string {
	if _, err := os.Stat(dir); err != nil {
		return fmt.Sprint("can't stat", dir, err.Error())
	}

	testFileName := randomFileName()
	testFile := filepath.Join(dir, testFileName)

	err := os.WriteFile(testFile, []byte(testFileName), 0600)
	if err != nil {
		return fmt.Sprint("can't write file in ", dir, err.Error())
	}

	rtest, err := os.ReadFile(testFile)
	if err != nil {
		return fmt.Sprint("can't read file ", testFileName, " in ", dir, err.Error())
	}

	if testFileName != string(rtest) {
		return fmt.Sprint("what we read from ", testFile, " is not what we wrote")
	}

	err = os.Remove(testFile)
	if err != nil {
		return fmt.Sprint("can't delete file ", testFileName, " in ", dir, err.Error())
	}

	return ""
}

func randomFileName() string {
	b := make([]byte, 15)
	if _, err := rand.Read(b); err != nil {
		panic("failed to read random bytes to test write-ability" + err.Error())
	}
	bhex := base32.StdEncoding.EncodeToString(b)
	return bhex + ".test"
}
