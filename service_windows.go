//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const crmServiceName = "CRM"

func handleWindowsServiceCommand(args []string) (bool, error) {
	if len(args) == 0 || args[0] != "--service" {
		return false, nil
	}
	if len(args) < 2 {
		return true, fmt.Errorf("usage: --service install|uninstall|start|stop|restart|status")
	}
	switch strings.ToLower(args[1]) {
	case "install":
		return true, installWindowsService()
	case "uninstall":
		return true, uninstallWindowsService()
	case "start":
		return true, startWindowsService()
	case "stop":
		return true, stopWindowsService()
	case "restart":
		return true, restartWindowsService()
	case "status":
		return true, statusWindowsService()
	default:
		return true, fmt.Errorf("unknown service action: %s", args[1])
	}
}

func shouldRunAsWindowsService() bool {
	interactive, err := svc.IsAnInteractiveSession()
	return err == nil && !interactive
}

func runWindowsService() error {
	return svc.Run(crmServiceName, &crmWindowsService{})
}

type crmWindowsService struct{}

func (s *crmWindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepts = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	done := make(chan struct{})
	go func() {
		if exe, err := serviceBinaryPath(); err == nil {
			_ = os.Chdir(filepath.Dir(exe))
		}
		runApp()
		close(done)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: accepts}

	for {
		select {
		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				changes <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				return false, 0
			}
		case <-done:
			return false, 0
		}
	}
}

func serviceBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}

func installWindowsService() error {
	exe, err := serviceBinaryPath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	if existing, err := m.OpenService(crmServiceName); err == nil {
		existing.Close()
		return fmt.Errorf("service already installed")
	}

	s, err := m.CreateService(crmServiceName, exe, mgr.Config{
		DisplayName:      "CRM",
		Description:      "CRM sales system",
		StartType:        mgr.StartAutomatic,
		DelayedAutoStart: true,
	})
	if err != nil {
		return err
	}
	defer s.Close()
	if err := s.Start(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "already running") {
		return err
	}
	return nil
}

func uninstallWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(crmServiceName)
	if err != nil {
		return err
	}
	defer s.Close()
	_, _ = s.Control(svc.Stop)
	if err := s.Delete(); err != nil {
		return err
	}
	return nil
}

func startWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(crmServiceName)
	if err != nil {
		return err
	}
	defer s.Close()

	return s.Start()
}

func stopWindowsService() error {
	return controlWindowsService(svc.Stop)
}

func restartWindowsService() error {
	if err := stopWindowsService(); err != nil {
		return err
	}
	return startWindowsService()
}

func statusWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(crmServiceName)
	if err != nil {
		return err
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return err
	}
	fmt.Printf("status: %v\n", status.State)
	return nil
}

func controlWindowsService(cmd svc.Cmd) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(crmServiceName)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = s.Control(cmd)
	return err
}
